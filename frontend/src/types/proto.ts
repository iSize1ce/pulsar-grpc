export interface ProtoField {
  name: string
  number?: number
  type: string
  repeated?: boolean
  optional?: boolean
  deprecated?: boolean
  messageType?: string
  fields?: ProtoField[]
  enumType?: string
  enumValues?: EnumValue[]
  oneofOptions?: ProtoField[]
  mapKey?: MapComponent
  mapValue?: MapComponent
}

export interface EnumValue {
  name: string
  number: number
}

export interface MapComponent {
  type: string
  messageType?: string
  fields?: ProtoField[]
  enumType?: string
  enumValues?: EnumValue[]
}

export interface WktConfig {
  label: string
  kind: string
  placeholder: string
}
