import type { ProtoField, MapComponent } from '@/types/proto'
import { wktScalarCfg } from '@/utils/wktScalar'

export function typeLabel(f: ProtoField): string {
  const pre = f.repeated ? '[]' : f.optional ? '?' : ''
  if (f.type === 'map') {
    const keyType = mapComponentTypeLabel(f.mapKey)
    const valueType = mapComponentTypeLabel(f.mapValue)
    return `${pre}map<${keyType}, ${valueType}>`
  }
  const wkt = wktScalarCfg(f)
  if (wkt) return pre + wkt.label
  if (f.type === 'message') return pre + (f.messageType?.split('.').pop() ?? 'message')
  if (f.type === 'oneof') return 'oneof'
  return pre + f.type
}

export function mapComponentTypeLabel(c: MapComponent | undefined): string {
  if (!c) return 'any'
  const wkt = wktScalarCfg(c)
  if (wkt) return wkt.label
  if (c.type === 'message') return c.messageType?.split('.').pop() ?? 'message'
  return c.type || 'any'
}
