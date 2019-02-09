package tfsdk

import (
	"fmt"
	"reflect"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// ---------------------------
// This file contains some helpers for some reflection-driven dynamic behaviors
// we need to do elsewhere in the SDK, in an attempt to keep the main SDK code
// relatively easy to read.
//
// There should be no exported symbols in this file.
// ---------------------------

var diagnosticsType = reflect.TypeOf(Diagnostics(nil))

// wrapSimpleFunction dynamically binds the given arguments to the given
// function, or returns a developer-oriented error describing why it cannot.
//
// If the requested call is valid, the result is a function that takes no
// arguments, executes the requested call, and returns any diagnostics that
// result.
//
// As a convenience, if the given function is nil then a no-op function will
// be returned, for the common situation where a dynamic function is optional.
func wrapSimpleFunction(f interface{}, args ...interface{}) (func() Diagnostics, error) {
	if f == nil {
		return func() Diagnostics {
			return nil
		}, nil
	}

	fv := reflect.ValueOf(f)
	if fv.Kind() != reflect.Func {
		return nil, fmt.Errorf("value is %s, not Func", fv.Kind().String())
	}

	ft := fv.Type()
	if ft.NumOut() != 1 || !ft.Out(0).AssignableTo(diagnosticsType) {
		return nil, fmt.Errorf("must return Diagnostics")
	}

	convArgs, forceDiags, err := prepareDynamicCallArgs(f, args...)
	if err != nil {
		return nil, err
	}

	return func() Diagnostics {
		if len(forceDiags) > 0 {
			return forceDiags
		}

		out := fv.Call(convArgs)
		return out[0].Interface().(Diagnostics)
	}, nil
}

func prepareDynamicCallArgs(f interface{}, args ...interface{}) ([]reflect.Value, Diagnostics, error) {
	fv := reflect.ValueOf(f)
	if fv.Kind() != reflect.Func {
		return nil, nil, fmt.Errorf("value is %s, not Func", fv.Kind().String())
	}

	ft := fv.Type()
	if got, want := ft.NumIn(), len(args); got != want {
		// (this error assumes that "args" is defined by the SDK code and thus
		// correct, while f comes from the provider and so is wrong.)
		return nil, nil, fmt.Errorf("should have %d arguments, but has %d", want, got)
	}

	var forceDiags Diagnostics

	convArgs := make([]reflect.Value, len(args))
	for i, rawArg := range args {
		wantType := ft.In(i)
		switch arg := rawArg.(type) {
		case cty.Value:
			// As a special case, we handle cty.Value arguments through gocty.
			targetVal := reflect.New(wantType)
			err := gocty.FromCtyValue(arg, targetVal.Interface())
			if err != nil {
				// While most of the errors in here are written as if the
				// f interface is wrong, for this particular case we invert
				// that to consider the f argument as a way to specify
				// constraints on the user-provided value. However, we don't
				// have much context here for what the wrapped function is for,
				// so our error message is necessarily generic. Providers should
				// generally not rely on this error form and should instead
				// ensure that all user-supplyable values can be accepted.
				forceDiags = forceDiags.Append(Diagnostic{
					Severity: Error,
					Summary:  "Invalid argument value",
					Detail:   fmt.Sprintf("Invalid value: %s.", FormatError(err)),
				})
			}
			convArgs[i] = targetVal.Elem() // New created a pointer, but we want the referent
		default:
			// All other arguments must be directly assignable.
			argVal := reflect.ValueOf(rawArg)
			if !argVal.Type().AssignableTo(wantType) {
				return nil, nil, fmt.Errorf("argument %d must accept %T", i, rawArg)
			}
			convArgs[i] = argVal
		}
	}

	return convArgs, forceDiags, nil
}
