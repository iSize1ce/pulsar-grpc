package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	// Register standard gRPC error detail types so protojson can resolve Any fields
	// like BadRequest, ErrorInfo, etc. Without this import, protojson.Marshal
	// would fail to serialize these well-known types inside grpc Status details.
	_ "google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/jhump/protoreflect/v2/protoresolve"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	reflectTimeout = 10 * time.Second
	invokeTimeout  = 30 * time.Second
)

// skipServices hides gRPC's own reflection services from the service list,
// since calling them directly doesn't make sense for end users.
var skipServices = map[string]struct{}{
	"grpc.reflection.v1.ServerReflection":      {},
	"grpc.reflection.v1alpha.ServerReflection": {},
}

// handleServices returns the list of gRPC services with their methods.
// Reflection-internal services are filtered out.
func handleServices(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := decodeBody(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, `{"error":"url is required"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), reflectTimeout)
	defer cancel()

	session, err := openSession(ctx, req.URL)
	if err != nil {
		slog.Warn("handleServices: openSession failed",
			"url", req.URL,
			"request_id", requestID(r.Context()),
			"err", err,
		)
		respondError(w, err)
		return
	}
	defer session.Close()

	services, err := session.client.ListServices()
	if err != nil {
		slog.Warn("handleServices: ListServices failed",
			"url", req.URL,
			"request_id", requestID(r.Context()),
			"err", err,
		)
		respondError(w, err)
		return
	}

	type methodInfo struct {
		Name            string `json:"name"`
		Input           string `json:"input"`
		Output          string `json:"output"`
		ClientStreaming bool   `json:"clientStreaming"`
		ServerStreaming bool   `json:"serverStreaming"`
	}
	type serviceInfo struct {
		Name    string       `json:"name"`
		Methods []methodInfo `json:"methods"`
	}

	result := make([]serviceInfo, 0, len(services))
	for _, svc := range services {
		name := string(svc)
		if _, skip := skipServices[name]; skip {
			continue
		}
		si := serviceInfo{Name: name}
		sd, err := findServiceDesc(session.resolver, name)
		if err != nil {
			slog.Warn("handleServices: findServiceDesc failed",
				"service", name,
				"request_id", requestID(r.Context()),
				"err", err,
			)
		} else {
			si.Methods = make([]methodInfo, sd.Methods().Len())
			for i := range si.Methods {
				md := sd.Methods().Get(i)
				si.Methods[i] = methodInfo{
					Name:            string(md.Name()),
					Input:           string(md.Input().FullName()),
					Output:          string(md.Output().FullName()),
					ClientStreaming: md.IsStreamingClient(),
					ServerStreaming: md.IsStreamingServer(),
				}
			}
		}
		result = append(result, si)
	}

	respondJSON(w, result)
}

// handleDescribe returns the input message schema for a given method.
// Used by the frontend to build the dynamic form.
func handleDescribe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL     string `json:"url"`
		Service string `json:"service"`
		Method  string `json:"method"`
	}
	if err := decodeBody(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.URL == "" || req.Service == "" || req.Method == "" {
		http.Error(w, `{"error":"url, service and method are required"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), reflectTimeout)
	defer cancel()

	session, err := openSession(ctx, req.URL)
	if err != nil {
		slog.Warn("handleDescribe: openSession failed",
			"url", req.URL,
			"service", req.Service,
			"method", req.Method,
			"request_id", requestID(r.Context()),
			"err", err,
		)
		respondError(w, err)
		return
	}
	defer session.Close()

	md, err := resolveMethod(session, req.Service, req.Method)
	if err != nil {
		slog.Warn("handleDescribe: resolveMethod failed",
			"url", req.URL,
			"service", req.Service,
			"method", req.Method,
			"request_id", requestID(r.Context()),
			"err", err,
		)
		respondError(w, err)
		return
	}

	respondJSON(w, describeMessage(md.Input()))
}

// handleInvoke executes a unary gRPC call and returns the response
// along with debug info (headers, trailers, timing).
func handleInvoke(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL     string          `json:"url"`
		Service string          `json:"service"`
		Method  string          `json:"method"`
		Payload json.RawMessage `json:"payload"`
		Meta    []metaEntry     `json:"meta"`
	}
	if err := decodeBody(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), invokeTimeout)
	defer cancel()

	// Attach outgoing metadata (gRPC headers) from the request.
	outMeta := buildOutgoingMeta(req.Meta)
	if len(outMeta) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, outMeta)
	}

	session, err := openSession(ctx, req.URL)
	if err != nil {
		slog.Warn("handleInvoke: openSession failed",
			"url", req.URL,
			"service", req.Service,
			"method", req.Method,
			"request_id", requestID(r.Context()),
			"err", err,
		)
		debug := map[string]any{"url": req.URL, "service": req.Service, "method": req.Method}
		respondJSON(w, buildErrorResponse(err, debug, nil))
		return
	}
	defer session.Close()

	md, err := resolveMethod(session, req.Service, req.Method)
	if err != nil {
		slog.Warn("handleInvoke: resolveMethod failed",
			"url", req.URL,
			"service", req.Service,
			"method", req.Method,
			"request_id", requestID(r.Context()),
			"err", err,
		)
		debug := map[string]any{"url": req.URL, "service": req.Service, "method": req.Method}
		respondJSON(w, buildErrorResponse(err, debug, session.resolver))
		return
	}

	// Deserialize the JSON payload into a dynamic protobuf message.
	reqMsg := dynamicpb.NewMessage(md.Input())
	if err := protojson.Unmarshal(req.Payload, reqMsg); err != nil {
		slog.Warn("handleInvoke: protojson.Unmarshal failed",
			"url", req.URL,
			"service", req.Service,
			"method", req.Method,
			"request_id", requestID(r.Context()),
			"err", err,
		)
		respondError(w, fmt.Errorf("invalid payload: %w", err))
		return
	}

	// Execute the gRPC call and collect response metadata.
	var respHeaders, respTrailers metadata.MD
	startTime := time.Now()

	respMsg, invokeErr := session.stub.InvokeRpc(ctx, md, reqMsg,
		grpc.Header(&respHeaders),
		grpc.Trailer(&respTrailers),
	)

	duration := time.Since(startTime)

	debug := buildDebugInfo(req.URL, req.Service, req.Method, req.Payload,
		outMeta, respHeaders, respTrailers, startTime, duration)

	if invokeErr != nil {
		slog.Info("handleInvoke: gRPC error",
			"url", req.URL,
			"service", req.Service,
			"method", req.Method,
			"duration_ms", duration.Milliseconds(),
			"request_id", requestID(r.Context()),
			"err", invokeErr,
		)
		respondJSON(w, buildErrorResponse(invokeErr, debug, session.resolver))
		return
	}

	debug["statusCode"] = "OK"
	protoResp, ok := respMsg.(proto.Message)
	if !ok {
		respondError(w, fmt.Errorf("unexpected response type %T", respMsg))
		return
	}

	data, err := marshalProtoMessageOrdered(protoResp.ProtoReflect().Descriptor(), protoResp.ProtoReflect())
	if err != nil {
		respondError(w, fmt.Errorf("marshal response: %w", err))
		return
	}
	if b, err := json.Marshal(data); err == nil {
		debug["responseSize"] = len(b)
	}

	slog.Info("handleInvoke: ok",
		"url", req.URL,
		"service", req.Service,
		"method", req.Method,
		"duration_ms", duration.Milliseconds(),
		"request_id", requestID(r.Context()),
	)

	type invokeResponse struct {
		Data  any            `json:"data"`
		Debug map[string]any `json:"debug"`
	}
	respondJSON(w, invokeResponse{Data: data, Debug: debug})
}

// resolveMethod finds a unary method descriptor by service and method name.
// Returns an error if the method uses streaming (not supported).
func resolveMethod(session *grpcSession, service, method string) (protoreflect.MethodDescriptor, error) {
	sd, err := findServiceDesc(session.resolver, service)
	if err != nil {
		return nil, err
	}
	md, err := findMethodDesc(sd, method)
	if err != nil {
		return nil, err
	}
	if md.IsStreamingClient() || md.IsStreamingServer() {
		return nil, fmt.Errorf("streaming methods are not supported")
	}
	return md, nil
}

// ─── handleInvoke helpers ────────────────────────────────────────────────────

// metaEntry represents a single key-value pair from the frontend metadata list.
type metaEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// buildOutgoingMeta converts frontend meta entries into gRPC metadata.
// Entries with empty keys are skipped.
func buildOutgoingMeta(entries []metaEntry) metadata.MD {
	pairs := make([]string, 0, len(entries)*2)
	for _, e := range entries {
		if e.Key != "" {
			pairs = append(pairs, e.Key, e.Value)
		}
	}
	if len(pairs) == 0 {
		return nil
	}
	return metadata.Pairs(pairs...)
}

// buildDebugInfo creates the debug section of the invoke response,
// containing timing, metadata, and request details.
func buildDebugInfo(
	reqURL, service, method string,
	payload json.RawMessage,
	outMeta, respHeaders, respTrailers metadata.MD,
	startTime time.Time, duration time.Duration,
) map[string]any {
	return map[string]any{
		"requestUrl":       reqURL,
		"method":           fmt.Sprintf("/%s/%s", service, method),
		"requestPayload":   json.RawMessage(payload),
		"requestMeta":      metadataToMap(outMeta),
		"responseHeaders":  metadataToMap(respHeaders),
		"responseTrailers": metadataToMap(respTrailers),
		"sentAt":           startTime.Format(time.RFC3339Nano),
		"receivedAt":       startTime.Add(duration).Format(time.RFC3339Nano),
		"duration":         duration.String(),
		"durationMs":       duration.Milliseconds(),
	}
}

// buildErrorResponse constructs the JSON response body when a gRPC call fails.
// Extracts gRPC status code, message, and any error details (like BadRequest violations).
func buildErrorResponse(invokeErr error, debug map[string]any, resolver protoresolve.Resolver) map[string]any {
	debug["error"] = invokeErr.Error()
	resp := map[string]any{"debug": debug}

	st := grpcStatus(invokeErr)

	debug["statusCode"] = st.Code().String()
	debug["statusMessage"] = st.Message()
	resp["grpcStatusCode"] = int(st.Code())
	resp["grpcStatus"] = st.Code().String()
	resp["grpcMessage"] = st.Message()

	protoDetails := st.Proto().GetDetails()
	if len(protoDetails) == 0 {
		return resp
	}

	detailsList := make([]any, 0, len(protoDetails))
	for _, a := range protoDetails {
		detail := map[string]any{"@type": a.GetTypeUrl()}
		if b, err := protojson.Marshal(a); err == nil {
			var parsed map[string]any
			if json.Unmarshal(b, &parsed) == nil {
				detail = parsed
			}
		} else if resolver != nil {
			// Type not in global registry — try resolving via gRPC reflection.
			if parsed := resolveAnyViaReflection(a, resolver); parsed != nil {
				detail = parsed
			}
		}
		detailsList = append(detailsList, detail)
	}
	resp["errorDetails"] = detailsList

	return resp
}

// grpcStatus extracts a gRPC status from an error, unwrapping wrapped errors if needed.
// Unlike status.FromError, this handles errors wrapped with fmt.Errorf("%w").
func grpcStatus(err error) *status.Status {
	for e := err; e != nil; e = errors.Unwrap(e) {
		if st, ok := status.FromError(e); ok {
			return st
		}
	}
	return status.Convert(err)
}

// resolveAnyViaReflection decodes a protobuf Any using the reflection-based resolver
// when the type is not in the global registry (e.g. custom server-specific error types).
func resolveAnyViaReflection(a proto.Message, resolver protoresolve.Resolver) map[string]any {
	ref := a.ProtoReflect()
	typeURL := ref.Get(ref.Descriptor().Fields().ByName("type_url")).String()
	valueBytes := ref.Get(ref.Descriptor().Fields().ByName("value")).Bytes()

	fullName := typeURL
	if i := strings.LastIndex(typeURL, "/"); i >= 0 {
		fullName = typeURL[i+1:]
	}

	desc, err := resolver.FindDescriptorByName(protoreflect.FullName(fullName))
	if err != nil {
		return nil
	}
	msgDesc, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil
	}

	dynMsg := dynamicpb.NewMessage(msgDesc)
	if err := proto.Unmarshal(valueBytes, dynMsg); err != nil {
		return nil
	}

	b, err := protojson.Marshal(dynMsg)
	if err != nil {
		return nil
	}
	var parsed map[string]any
	if json.Unmarshal(b, &parsed) != nil {
		return nil
	}
	fixDynamicJSON(msgDesc, dynMsg.ProtoReflect(), parsed)
	parsed["@type"] = typeURL
	return parsed
}

// fixDynamicJSON patches JSON output from protojson.Marshal on dynamicpb messages:
//   - Adds default values for missing scalar fields (protojson omits zero values)
//   - Resolves enum fields to string names (protojson writes numbers for unregistered enums)
//   - Recurses into nested messages, repeated fields, and map values
func fixDynamicJSON(md protoreflect.MessageDescriptor, msg protoreflect.Message, out map[string]any) {
	for i := 0; i < md.Fields().Len(); i++ {
		fd := md.Fields().Get(i)
		name := fd.JSONName()

		if fd.IsList() {
			arr, ok := out[name].([]any)
			if !ok {
				out[name] = []any{}
				continue
			}
			if fd.Kind() == protoreflect.EnumKind {
				list := msg.Get(fd).List()
				for j := 0; j < len(arr) && j < list.Len(); j++ {
					num := list.Get(j).Enum()
					if ev := fd.Enum().Values().ByNumber(num); ev != nil {
						arr[j] = string(ev.Name())
					}
				}
			} else if fd.Kind() == protoreflect.MessageKind {
				list := msg.Get(fd).List()
				for j := 0; j < len(arr) && j < list.Len(); j++ {
					if nested, ok := arr[j].(map[string]any); ok {
						fixDynamicJSON(fd.Message(), list.Get(j).Message(), nested)
					}
				}
			}
			continue
		}

		if fd.IsMap() {
			m, ok := out[name].(map[string]any)
			if !ok {
				out[name] = map[string]any{}
				continue
			}
			valFd := fd.MapValue()
			if valFd.Kind() == protoreflect.EnumKind {
				mapVal := msg.Get(fd).Map()
				mapVal.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
					key := fmt.Sprint(k.Interface())
					num := v.Enum()
					if ev := valFd.Enum().Values().ByNumber(num); ev != nil {
						m[key] = string(ev.Name())
					}
					return true
				})
			} else if valFd.Kind() == protoreflect.MessageKind {
				mapVal := msg.Get(fd).Map()
				mapVal.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
					key := fmt.Sprint(k.Interface())
					if nested, ok := m[key].(map[string]any); ok {
						fixDynamicJSON(valFd.Message(), v.Message(), nested)
					}
					return true
				})
			}
			continue
		}

		if fd.Kind() == protoreflect.EnumKind {
			num := msg.Get(fd).Enum()
			if ev := fd.Enum().Values().ByNumber(num); ev != nil {
				out[name] = string(ev.Name())
			} else if _, exists := out[name]; !exists {
				out[name] = 0
			}
			continue
		}

		if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
			if msg.Has(fd) {
				if nested, ok := out[name].(map[string]any); ok {
					fixDynamicJSON(fd.Message(), msg.Get(fd).Message(), nested)
				}
			}
			continue
		}

		if _, exists := out[name]; exists {
			continue
		}
		switch fd.Kind() {
		case protoreflect.BoolKind:
			out[name] = false
		case protoreflect.StringKind:
			out[name] = ""
		case protoreflect.BytesKind:
			out[name] = ""
		default:
			out[name] = 0
		}
	}
}

// metadataToMap converts gRPC metadata into a plain map for JSON serialization.
// Single-value keys are stored as strings; multi-value keys as string arrays.
func metadataToMap(md metadata.MD) map[string]any {
	result := make(map[string]any, len(md))
	for k, vals := range md {
		if len(vals) == 1 {
			result[k] = vals[0]
		} else {
			result[k] = vals
		}
	}
	return result
}

// ─── Ordered JSON serialization ──────────────────────────────────────────────
//
// protojson emits fields in descriptor order and proto3 default values are
// omitted. We re-implement serialization to:
//   - preserve the proto field order (important for UX consistency)
//   - always emit explicit defaults for every field
//   - represent int64/uint64 as decimal strings (JSON number limits)

type orderedField struct {
	key   string
	value any
}

type orderedJSONObject []orderedField

// MarshalJSON writes object keys in the exact order stored in the slice.
func (o orderedJSONObject) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, f := range o {
		if i > 0 {
			b.WriteByte(',')
		}
		k, err := json.Marshal(f.key)
		if err != nil {
			return nil, err
		}
		v, err := json.Marshal(f.value)
		if err != nil {
			return nil, err
		}
		b.Write(k)
		b.WriteByte(':')
		b.Write(v)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// marshalProtoMessageOrdered serializes a protobuf message to JSON-compatible data,
// preserving proto field order and emitting explicit defaults.
func marshalProtoMessageOrdered(md protoreflect.MessageDescriptor, msg protoreflect.Message) (orderedJSONObject, error) {
	ordered := make(orderedJSONObject, 0, md.Fields().Len())
	for i := 0; i < md.Fields().Len(); i++ {
		fd := md.Fields().Get(i)
		has := msg.Has(fd)
		od := fd.ContainingOneof()

		// Real oneof fields must stay omitted unless selected.
		if od != nil && !od.IsSynthetic() && !has {
			continue
		}

		var (
			value any
			err   error
		)

		// Proto3 optional (synthetic oneof) that's not present → explicit null.
		if od != nil && od.IsSynthetic() && !has {
			value = nil
		} else if has {
			value, err = presentFieldJSONValue(fd, msg.Get(fd))
		} else {
			value, err = defaultFieldJSONValue(fd)
		}
		if err != nil {
			return nil, err
		}

		ordered = append(ordered, orderedField{key: fd.JSONName(), value: value})
	}
	return ordered, nil
}

func presentFieldJSONValue(fd protoreflect.FieldDescriptor, v protoreflect.Value) (any, error) {
	if fd.IsList() {
		list := v.List()
		out := make([]any, list.Len())
		for i := range out {
			elem, err := scalarOrMessageJSONValue(fd, list.Get(i), true)
			if err != nil {
				return nil, err
			}
			out[i] = elem
		}
		return out, nil
	}

	if fd.IsMap() {
		m := v.Map()
		out := make(map[string]any, m.Len())
		valFd := fd.MapValue()
		var convErr error
		m.Range(func(k protoreflect.MapKey, mv protoreflect.Value) bool {
			key, err := mapKeyToJSONKey(fd.MapKey(), k)
			if err != nil {
				convErr = err
				return false
			}
			converted, err := scalarOrMessageJSONValue(valFd, mv, false)
			if err != nil {
				convErr = err
				return false
			}
			out[key] = converted
			return true
		})
		if convErr != nil {
			return nil, convErr
		}
		return out, nil
	}

	return scalarOrMessageJSONValue(fd, v, false)
}

func defaultFieldJSONValue(fd protoreflect.FieldDescriptor) (any, error) {
	if fd.IsList() {
		return []any{}, nil
	}
	if fd.IsMap() {
		return map[string]any{}, nil
	}
	if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
		return nil, nil
	}
	return scalarDefaultJSONValue(fd), nil
}

func scalarOrMessageJSONValue(fd protoreflect.FieldDescriptor, v protoreflect.Value, listElem bool) (any, error) {
	if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
		msg := v.Message()
		if isWellKnownJSONType(fd.Message()) {
			return marshalProtoToAny(msg.Interface())
		}
		return marshalProtoMessageOrdered(fd.Message(), msg)
	}
	return scalarJSONValue(fd, v, listElem), nil
}

func scalarDefaultJSONValue(fd protoreflect.FieldDescriptor) any {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return false
	case protoreflect.StringKind:
		return ""
	case protoreflect.BytesKind:
		return ""
	case protoreflect.EnumKind:
		num := fd.Default().Enum()
		if ev := fd.Enum().Values().ByNumber(num); ev != nil {
			return string(ev.Name())
		}
		return int32(num)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "0"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "0"
	default:
		return 0
	}
}

func scalarJSONValue(fd protoreflect.FieldDescriptor, v protoreflect.Value, listElem bool) any {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return v.Bool()
	case protoreflect.StringKind:
		return v.String()
	case protoreflect.BytesKind:
		return base64.StdEncoding.EncodeToString(v.Bytes())
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return int32(v.Int())
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return uint32(v.Uint())
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return strconv.FormatInt(v.Int(), 10)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return strconv.FormatUint(v.Uint(), 10)
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		f := v.Float()
		if math.IsNaN(f) {
			return "NaN"
		}
		if math.IsInf(f, 1) {
			return "Infinity"
		}
		if math.IsInf(f, -1) {
			return "-Infinity"
		}
		if fd.Kind() == protoreflect.FloatKind {
			return float32(f)
		}
		return f
	case protoreflect.EnumKind:
		num := v.Enum()
		if ev := fd.Enum().Values().ByNumber(num); ev != nil {
			return string(ev.Name())
		}
		return int32(num)
	default:
		return nil
	}
}

func mapKeyToJSONKey(kd protoreflect.FieldDescriptor, k protoreflect.MapKey) (string, error) {
	switch kd.Kind() {
	case protoreflect.StringKind:
		return k.String(), nil
	case protoreflect.BoolKind:
		if k.Bool() {
			return "true", nil
		}
		return "false", nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return strconv.FormatInt(k.Int(), 10), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return strconv.FormatUint(k.Uint(), 10), nil
	default:
		return "", fmt.Errorf("unsupported map key kind: %s", kd.Kind())
	}
}

func marshalProtoToAny(msg protoreflect.ProtoMessage) (any, error) {
	b, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(msg)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func isWellKnownJSONType(md protoreflect.MessageDescriptor) bool {
	switch string(md.FullName()) {
	case "google.protobuf.Any",
		"google.protobuf.Timestamp",
		"google.protobuf.Duration",
		"google.protobuf.FieldMask",
		"google.protobuf.Struct",
		"google.protobuf.ListValue",
		"google.protobuf.Value",
		"google.protobuf.DoubleValue",
		"google.protobuf.FloatValue",
		"google.protobuf.Int64Value",
		"google.protobuf.UInt64Value",
		"google.protobuf.Int32Value",
		"google.protobuf.UInt32Value",
		"google.protobuf.BoolValue",
		"google.protobuf.StringValue",
		"google.protobuf.BytesValue":
		return true
	default:
		return false
	}
}
