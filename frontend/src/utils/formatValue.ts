import type { ProtoField, MapComponent } from '@/types/proto'
import { wktScalarCfg } from './wktScalar'
import { INT_TYPES, UINT_TYPES, isNumericProtoType } from './protoTypes'

function numericFieldType(f: ProtoField | MapComponent): string {
  const cfg = wktScalarCfg(f)
  return cfg ? cfg.kind : f.type
}

export function isIntegerLikeField(f: ProtoField | MapComponent | null | undefined): boolean {
  if (!f) return false
  const cfg = wktScalarCfg(f)
  if (cfg) return ['int32', 'int64', 'uint32', 'uint64'].includes(cfg.kind)
  return INT_TYPES.includes(f.type) || UINT_TYPES.includes(f.type)
}

export function formatFieldValueForInput(v: any, f: ProtoField | MapComponent): string {
  if (v == null) return ''
  const cfg = wktScalarCfg(f)
  if (cfg && ['int32', 'int64', 'uint32', 'uint64'].includes(cfg.kind) && Number(v) === 0) {
    return ''
  }
  if (cfg && (cfg.kind === 'jsonObject' || cfg.kind === 'jsonArray' || cfg.kind === 'jsonValue')) {
    return typeof v === 'string' ? v : JSON.stringify(v)
  }
  return String(v)
}

export function formatOneofValueForInput(v: any, f: ProtoField | MapComponent): string {
  if (v == null) return ''
  const t = numericFieldType(f)
  if (isNumericProtoType(t) && Number(v) === 0) return ''
  return formatFieldValueForInput(v, f)
}

export function formatCollectionValueForInput(v: any, f: ProtoField | MapComponent): string {
  if (v == null) return ''
  if (isIntegerLikeField(f) && Number(v) === 0) return ''
  return formatFieldValueForInput(v, f)
}
