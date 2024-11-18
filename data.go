package newsdoc

// Get the value with the given key. This is safe to use on nil DataMaps.
func (bd DataMap) Get(key string, defaultValue string) string {
	if bd == nil {
		return defaultValue
	}

	v, ok := bd[key]
	if !ok {
		return defaultValue
	}

	return v
}

// Delete the values with the given keys. This is safe to use on nil DataMaps.
func (bd DataMap) Delete(keys ...string) {
	if bd == nil {
		return
	}

	for _, key := range keys {
		delete(bd, key)
	}
}

// DropEmpty removes all entries with empty values. This is safe to use on nil
// DataMaps.
func (bd DataMap) DropEmpty() {
	if bd == nil {
		return
	}

	for k, v := range bd {
		if v != "" {
			continue
		}

		delete(bd, k)
	}
}

// UpsertData adds the values from new into data. If data is nil a new DataMap
// will be created.
func UpsertData(data DataMap, newData DataMap) DataMap {
	if data == nil {
		data = make(DataMap)
	}

	for k, v := range newData {
		data[k] = v
	}

	return data
}

// WithDefaults adds the values from defaults into data if the vallue for
// corresponding key is unset or empty. If data is nil a new DataMap will be
// created.
func DataWithDefaults(data DataMap, defaults DataMap) DataMap {
	if data == nil {
		data = make(DataMap)
	}

	for k, v := range defaults {
		if data[k] == "" {
			data[k] = v
		}
	}

	return data
}

// CopyData copies the given keys from the source data map to the
// destination. Keys will only be copied if they actually exists and it's safe
// to call the function with nil DataMaps. The result will always be a non-nil
// DataMap.
func CopyData(dst DataMap, src DataMap, keys ...string) DataMap {
	if dst == nil {
		dst = make(DataMap)
	}

	if src == nil {
		return dst
	}

	for _, k := range keys {
		v, ok := src[k]
		if !ok {
			continue
		}

		dst[k] = v
	}

	return dst
}
