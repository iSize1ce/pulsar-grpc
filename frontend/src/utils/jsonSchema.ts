import type { ProtoField, MapComponent } from '@/types/proto'
import { wktScalarCfg } from './wktScalar'
import { toCamelCase } from './camelCase'

const MAX_SCHEMA_DEPTH = 5

type GetMessageFields = (f: ProtoField | MapComponent) => ProtoField[]

export function protoFieldsToJsonSchema(
  fields: ProtoField[] | undefined,
  getMessageFields: GetMessageFields,
  seen?: Set<string>,
  depth?: number,
): any {
  if (!fields?.length || (depth ?? 0) > MAX_SCHEMA_DEPTH) return { type: 'object' }
  seen = seen || new Set()
  depth = depth || 0
  const properties: Record<string, any> = {}
  for (const f of fields) {
    if (f.type === 'oneof' && f.oneofOptions) {
      for (const opt of f.oneofOptions) {
        properties[toCamelCase(opt.name)] = fieldToSchema(opt, getMessageFields, seen, depth)
      }
      continue
    }
    properties[toCamelCase(f.name)] = fieldToSchema(f, getMessageFields, seen, depth)
  }
  return { type: 'object', properties }
}

function fieldToSchema(
  f: ProtoField,
  getMessageFields: GetMessageFields,
  seen: Set<string>,
  depth: number,
): any {
  if (f.type === 'map') return mapSchema(f, getMessageFields, seen, depth)
  const base = scalarSchema(f, getMessageFields, seen, depth)
  return f.repeated ? { type: 'array', items: base } : base
}

function mapSchema(
  f: ProtoField,
  getMessageFields: GetMessageFields,
  seen: Set<string>,
  depth: number,
): any {
  return {
    type: 'object',
    additionalProperties: scalarSchema(
      (f.mapValue || { type: 'string' }) as ProtoField,
      getMessageFields,
      seen,
      depth,
    ),
  }
}

function scalarSchema(
  f: ProtoField | MapComponent,
  getMessageFields: GetMessageFields,
  seen: Set<string>,
  depth: number,
): any {
  const wkt = wktScalarCfg(f)
  if (wkt) {
    if (['timestamp', 'string', 'bytes'].includes(wkt.kind)) return { type: 'string' }
    if (wkt.kind === 'bool') return { type: 'boolean' }
    if (['int32', 'int64', 'uint32', 'uint64'].includes(wkt.kind)) return { type: 'integer' }
    if (['float', 'double'].includes(wkt.kind)) return { type: 'number' }
    if (wkt.kind === 'jsonArray') return { type: 'array' }
    if (wkt.kind === 'jsonValue') return {}
    return { type: 'object' }
  }
  switch (f.type) {
    case 'message': {
      const key = ('messageType' in f ? f.messageType : '') || ''
      if (key && seen.has(key)) return { type: 'object' }
      const next = new Set(seen)
      if (key) next.add(key)
      return protoFieldsToJsonSchema(
        getMessageFields(f as ProtoField),
        getMessageFields,
        next,
        depth + 1,
      )
    }
    case 'enum':
      return {
        type: 'string',
        enum: (('enumValues' in f ? f.enumValues : undefined) || []).map((v) => v.name),
      }
    case 'bool':
      return { type: 'boolean' }
    case 'string':
    case 'bytes':
      return { type: 'string' }
    case 'int32':
    case 'int64':
    case 'uint32':
    case 'uint64':
    case 'sint32':
    case 'sint64':
    case 'fixed32':
    case 'fixed64':
    case 'sfixed32':
    case 'sfixed64':
      return { type: 'integer' }
    case 'float':
    case 'double':
      return { type: 'number' }
    default:
      return {}
  }
}
