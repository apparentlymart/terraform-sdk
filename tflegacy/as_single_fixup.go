package tflegacy

import (
	"sort"
	"strings"
)

// FixupAsSingleResourceConfigIn modifies the given ResourceConfig in-place if
// any attributes in the schema have the AsSingle flag set, wrapping the given
// values for these in an extra level of slice so that they can be understood
// by legacy SDK code that'll be expecting to decode into a list/set.
func FixupAsSingleResourceConfigIn(rc *ResourceConfig, s map[string]*Schema) {
	if rc == nil {
		return
	}
	FixupAsSingleConfigValueIn(rc.Config, s)
}

// FixupAsSingleInstanceStateIn modifies the given InstanceState in-place if
// any attributes in the schema have the AsSingle flag set, adding additional
// index steps to the flatmap keys for these so that they can be understood
// by legacy SDK code that'll be expecting to decode into a list/set.
func FixupAsSingleInstanceStateIn(is *InstanceState, r *Resource) {
	fixupAsSingleInstanceState(is, r.Schema, "", fixupAsSingleFlatmapKeysIn)
}

// FixupAsSingleInstanceStateOut modifies the given InstanceState in-place if
// any attributes in the schema have the AsSingle flag set, removing unneeded
// index steps from the flatmap keys for these so that they can be understood
// by the shim back to Terraform Core as a single nested value.
func FixupAsSingleInstanceStateOut(is *InstanceState, r *Resource) {
	fixupAsSingleInstanceState(is, r.Schema, "", fixupAsSingleFlatmapKeysOut)
}

// FixupAsSingleInstanceDiffIn modifies the given InstanceDiff in-place if any
// attributes in the schema have the AsSingle flag set, adding additional index
// steps to the flatmap keys for these so that they can be understood by legacy
// SDK code that'll be expecting to decode into a list/set.
func FixupAsSingleInstanceDiffIn(id *InstanceDiff, r *Resource) {
	fixupAsSingleInstanceDiff(id, r.Schema, "", fixupAsSingleAttrsMapKeysIn)
}

// FixupAsSingleInstanceDiffOut modifies the given InstanceDiff in-place if any
// attributes in the schema have the AsSingle flag set, removing unneeded index
// steps from the flatmap keys for these so that they can be understood by the
// shim back to Terraform Core as a single nested value.
func FixupAsSingleInstanceDiffOut(id *InstanceDiff, r *Resource) {
	fixupAsSingleInstanceDiff(id, r.Schema, "", fixupAsSingleAttrsMapKeysOut)
}

// FixupAsSingleConfigValueIn modifies the given "config value" in-place if
// any attributes in the schema have the AsSingle flag set, wrapping the given
// values for these in an extra level of slice so that they can be understood
// by legacy SDK code that'll be expecting to decode into a list/set.
//
// "Config value" for the purpose of this function has the same meaning as for
// the hcl2shims: a map[string]interface{} using the same subset of Go value
// types that would be generated by HCL/HIL when decoding a configuration in
// Terraform v0.11.
func FixupAsSingleConfigValueIn(c map[string]interface{}, s map[string]*Schema) {
	for k, as := range s {
		if !as.AsSingle {
			continue // Don't touch non-AsSingle values at all. This is explicitly opt-in.
		}

		v, ok := c[k]
		if ok {
			c[k] = []interface{}{v}
		}

		if nr, ok := as.Elem.(*Resource); ok {
			// Recursively fixup nested attributes too
			nm, ok := v.(map[string]interface{})
			if !ok {
				// Weird for a nested resource to not be a map, but we'll tolerate it rather than crashing
				continue
			}
			FixupAsSingleConfigValueIn(nm, nr.Schema)
		}
	}
}

// FixupAsSingleConfigValueOut modifies the given "config value" in-place if
// any attributes in the schema have the AsSingle flag set, unwrapping the
// given values from their single-element slices so that they can be understood
// as a single object value by Terraform Core.
//
// This is the opposite of fixupAsSingleConfigValueIn.
func FixupAsSingleConfigValueOut(c map[string]interface{}, s map[string]*Schema) {
	for k, as := range s {
		if !as.AsSingle {
			continue // Don't touch non-AsSingle values at all. This is explicitly opt-in.
		}

		sv, ok := c[k].([]interface{})
		if ok && len(sv) != 0 { // Should always be a single-element slice, but if not we'll just leave it alone rather than crashing
			c[k] = sv[0]
			if nr, ok := as.Elem.(*Resource); ok {
				// Recursively fixup nested attributes too
				nm, ok := sv[0].(map[string]interface{})
				if ok {
					FixupAsSingleConfigValueOut(nm, nr.Schema)
				}
			}
		}
	}
}

func fixupAsSingleInstanceState(is *InstanceState, s map[string]*Schema, prefix string, fn func(map[string]string, string) string) {
	if is == nil {
		return
	}

	for k, as := range s {
		if !as.AsSingle {
			continue // Don't touch non-AsSingle values at all. This is explicitly opt-in.
		}

		nextPrefix := fn(is.Attributes, prefix+k+".")
		if nr, ok := as.Elem.(*Resource); ok {
			// Recursively fixup nested attributes too
			fixupAsSingleInstanceState(is, nr.Schema, nextPrefix, fn)
		}
	}
}

func fixupAsSingleInstanceDiff(id *InstanceDiff, s map[string]*Schema, prefix string, fn func(map[string]*ResourceAttrDiff, string) string) {
	if id == nil {
		return
	}

	for k, as := range s {
		if !as.AsSingle {
			continue // Don't touch non-AsSingle values at all. This is explicitly opt-in.
		}

		nextPrefix := fn(id.Attributes, prefix+k+".")
		if nr, ok := as.Elem.(*Resource); ok {
			// Recursively fixup nested attributes too
			fixupAsSingleInstanceDiff(id, nr.Schema, nextPrefix, fn)
		}
	}
}

// fixupAsSingleFlatmapKeysIn searches the given flatmap for all keys with
// the given prefix (which must end with a dot) and replaces them with keys
// where that prefix is followed by the dummy index "0." and, if any such
// keys are found, a ".#"-suffixed key is also added whose value is "1".
//
// This function will also replace an exact match of the given prefix with
// the trailing dot removed, to recognize values of primitive-typed attributes.
func fixupAsSingleFlatmapKeysIn(attrs map[string]string, prefix string) string {
	ks := make([]string, 0, len(attrs))
	for k := range attrs {
		ks = append(ks, k)
	}
	sort.Strings(ks) // Makes no difference for valid input, but will ensure we handle invalid input deterministically

	for _, k := range ks {
		newK, countK := fixupAsSingleFlatmapKeyIn(k, prefix)
		if _, exists := attrs[newK]; k != newK && !exists {
			attrs[newK] = attrs[k]
			delete(attrs, k)
		}
		if _, exists := attrs[countK]; countK != "" && !exists {
			attrs[countK] = "1"
		}
	}

	return prefix + "0."
}

// fixupAsSingleAttrsMapKeysIn searches the given AttrDiff map for all keys with
// the given prefix (which must end with a dot) and replaces them with keys
// where that prefix is followed by the dummy index "0." and, if any such
// keys are found, a ".#"-suffixed key is also added whose value is "1".
//
// This function will also replace an exact match of the given prefix with
// the trailing dot removed, to recognize values of primitive-typed attributes.
func fixupAsSingleAttrsMapKeysIn(attrs map[string]*ResourceAttrDiff, prefix string) string {
	ks := make([]string, 0, len(attrs))
	for k := range attrs {
		ks = append(ks, k)
	}
	sort.Strings(ks) // Makes no difference for valid input, but will ensure we handle invalid input deterministically

	for _, k := range ks {
		newK, countK := fixupAsSingleFlatmapKeyIn(k, prefix)
		if _, exists := attrs[newK]; k != newK && !exists {
			attrs[newK] = attrs[k]
			delete(attrs, k)
		}
		if _, exists := attrs[countK]; countK != "" && !exists {
			attrs[countK] = &ResourceAttrDiff{
				Old: "1", // One should _always_ be present, so this seems okay?
				New: "1",
			}
		}
	}

	return prefix + "0."
}

func fixupAsSingleFlatmapKeyIn(k, prefix string) (string, string) {
	exact := prefix[:len(prefix)-1]

	switch {
	case k == exact:
		return exact + ".0", exact + ".#"
	case strings.HasPrefix(k, prefix):
		return prefix + "0." + k[len(prefix):], prefix + "#"
	default:
		return k, ""
	}
}

// fixupAsSingleFlatmapKeysOut searches the given flatmap for all keys with
// the given prefix (which must end with a dot) and replaces them with keys
// where the following dot-separated label is removed, under the assumption that
// it's an index that is no longer needed and, if such a key is present, also
// remove the "count" key for the prefix, which is the prefix followed by "#".
func fixupAsSingleFlatmapKeysOut(attrs map[string]string, prefix string) string {
	ks := make([]string, 0, len(attrs))
	for k := range attrs {
		ks = append(ks, k)
	}
	sort.Strings(ks) // Makes no difference for valid input, but will ensure we handle invalid input deterministically

	for _, k := range ks {
		newK := fixupAsSingleFlatmapKeyOut(k, prefix)
		if newK != k && newK == "" {
			delete(attrs, k)
		} else if _, exists := attrs[newK]; newK != k && !exists {
			attrs[newK] = attrs[k]
			delete(attrs, k)
		}
	}

	delete(attrs, prefix+"#") // drop the count key, if it's present
	return prefix
}

// fixupAsSingleAttrsMapKeysOut searches the given AttrDiff map for all keys with
// the given prefix (which must end with a dot) and replaces them with keys
// where the following dot-separated label is removed, under the assumption that
// it's an index that is no longer needed and, if such a key is present, also
// remove the "count" key for the prefix, which is the prefix followed by "#".
func fixupAsSingleAttrsMapKeysOut(attrs map[string]*ResourceAttrDiff, prefix string) string {
	ks := make([]string, 0, len(attrs))
	for k := range attrs {
		ks = append(ks, k)
	}
	sort.Strings(ks) // Makes no difference for valid input, but will ensure we handle invalid input deterministically

	for _, k := range ks {
		newK := fixupAsSingleFlatmapKeyOut(k, prefix)
		if newK != k && newK == "" {
			delete(attrs, k)
		} else if _, exists := attrs[newK]; newK != k && !exists {
			attrs[newK] = attrs[k]
			delete(attrs, k)
		}
	}

	delete(attrs, prefix+"#") // drop the count key, if it's present
	return prefix
}

func fixupAsSingleFlatmapKeyOut(k, prefix string) string {
	if strings.HasPrefix(k, prefix) {
		remain := k[len(prefix):]
		if remain == "#" {
			// Don't need the count element anymore
			return ""
		}
		dotIdx := strings.Index(remain, ".")
		if dotIdx == -1 {
			return prefix[:len(prefix)-1] // no follow-on attributes then
		} else {
			return prefix + remain[dotIdx+1:] // everything after the next dot
		}
	}
	return k
}
