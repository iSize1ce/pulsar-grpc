import type { ProtoField, MapComponent } from '@/types/proto'
import { wktScalarCfg } from './wktScalar'

export function parseTimestampInput(v: any): string | undefined {
  if (v == null) return undefined
  const s = String(v).trim()
  if (!s) return undefined
  if (/^-?\d+$/.test(s)) {
    const d = new Date(Number(s) * 1000)
    if (!isNaN(d.getTime())) return d.toISOString()
  }
  return s
}

export function parseJsonLiteralInput(v: any): any {
  if (v == null) return undefined
  const s = String(v).trim()
  if (!s) return undefined
  try {
    return JSON.parse(s)
  } catch {
    return s
  }
}

export function parseVal(v: any, type: string): any {
  if (v === '' || v == null) return undefined
  if (
    [
      'int32',
      'int64',
      'uint32',
      'uint64',
      'sint32',
      'sint64',
      'fixed32',
      'fixed64',
      'sfixed32',
      'sfixed64',
    ].includes(type)
  ) {
    const n = parseInt(v, 10)
    return isNaN(n) ? v : n
  }
  if (type === 'float' || type === 'double') {
    const n = parseFloat(v)
    return isNaN(n) ? v : n
  }
  if (type === 'bool') return v === 'true'
  return v
}

export function parseFieldVal(v: any, f: ProtoField | MapComponent): any {
  const cfg = wktScalarCfg(f)
  if (cfg) {
    switch (cfg.kind) {
      case 'timestamp':
        return parseTimestampInput(v)
      case 'string':
      case 'bytes':
        return v === '' ? '' : parseVal(v, cfg.kind)
      case 'jsonObject':
      case 'jsonArray':
      case 'jsonValue':
        return parseJsonLiteralInput(v)
      default:
        return parseVal(v, cfg.kind)
    }
  }
  return parseVal(v, f.type)
}

export function zeroVal(type: string): any {
  if (type === 'bool') return false
  if (
    [
      'int32',
      'int64',
      'uint32',
      'uint64',
      'sint32',
      'sint64',
      'fixed32',
      'fixed64',
      'sfixed32',
      'sfixed64',
      'float',
      'double',
    ].includes(type)
  )
    return 0
  return ''
}

export function oneofDefaultValue(f: ProtoField | MapComponent | null | undefined): any {
  if (!f) return ''
  const cfg = wktScalarCfg(f)
  if (cfg) {
    if (cfg.kind === 'timestamp') return null
    if ('messageType' in f && f.messageType === 'google.protobuf.Duration') return null
    if (cfg.kind === 'jsonObject') return {}
    if (cfg.kind === 'jsonArray') return []
    if (cfg.kind === 'jsonValue') return ''
    return zeroVal(cfg.kind)
  }
  if (f.type === 'enum') {
    const values = ('enumValues' in f ? f.enumValues : undefined) || []
    const zero = values.find((v) => v.number === 0)
    if (zero) return zero.name
    if (values.length) return values[0].name
    return ''
  }
  return zeroVal(f.type)
}
