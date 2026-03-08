import type { ProtoField, MapComponent, WktConfig } from '@/types/proto'

const WKT_SCALAR: Record<string, WktConfig> = {
  'google.protobuf.Timestamp': {
    label: 'Timestamp',
    kind: 'timestamp',
    placeholder: 'RFC3339 or unix seconds',
  },
  'google.protobuf.Duration': { label: 'Duration', kind: 'string', placeholder: 'e.g. 123.456s' },
  'google.protobuf.FieldMask': {
    label: 'FieldMask',
    kind: 'string',
    placeholder: 'fooBar,baz.qux',
  },
  'google.protobuf.StringValue': { label: 'StringValue', kind: 'string', placeholder: 'string' },
  'google.protobuf.BoolValue': { label: 'BoolValue', kind: 'bool', placeholder: 'bool' },
  'google.protobuf.Int32Value': { label: 'Int32Value', kind: 'int32', placeholder: 'int32' },
  'google.protobuf.Int64Value': { label: 'Int64Value', kind: 'int64', placeholder: 'int64' },
  'google.protobuf.UInt32Value': { label: 'UInt32Value', kind: 'uint32', placeholder: 'uint32' },
  'google.protobuf.UInt64Value': { label: 'UInt64Value', kind: 'uint64', placeholder: 'uint64' },
  'google.protobuf.FloatValue': { label: 'FloatValue', kind: 'float', placeholder: 'float' },
  'google.protobuf.DoubleValue': { label: 'DoubleValue', kind: 'double', placeholder: 'double' },
  'google.protobuf.BytesValue': { label: 'BytesValue', kind: 'bytes', placeholder: 'base64' },
  'google.protobuf.Any': { label: 'Any', kind: 'jsonObject', placeholder: '{}' },
  'google.protobuf.Struct': { label: 'Struct', kind: 'jsonObject', placeholder: '{"key":"value"}' },
  'google.protobuf.ListValue': {
    label: 'ListValue',
    kind: 'jsonArray',
    placeholder: '["a",1,true]',
  },
  'google.protobuf.Value': {
    label: 'Value',
    kind: 'jsonValue',
    placeholder: '"text" | 1 | true | null | {} | []',
  },
}

export function wktScalarCfg(f: ProtoField | MapComponent | null | undefined): WktConfig | null {
  if (!f || f.type !== 'message' || !('messageType' in f) || !f.messageType) return null
  return WKT_SCALAR[f.messageType] || null
}

export function isWktScalarField(f: ProtoField | MapComponent | null | undefined): boolean {
  return !!wktScalarCfg(f)
}
