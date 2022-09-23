package test

import (
	"reflect"
	"testing"
)

func AssertEqual(t *testing.T, a, b interface{}) {
	t.Helper()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Not Equal. %v %v", a, b)
	}
}

func AssertNotEqual(t *testing.T, a, b interface{}) {
	t.Helper()
	if reflect.DeepEqual(a, b) {
		t.Errorf("Equal. %v %v", a, b)
	}
}

func AssertTrue(t *testing.T, v bool) {
	t.Helper()
	if !v {
		t.Errorf("Not True.")
	}
}

func AssertFalse(t *testing.T, v bool) {
	t.Helper()
	if v {
		t.Errorf("Not False.")
	}
}

func AssertNotNil(t *testing.T, v interface{}) {
	t.Helper()
	if v == nil {
		t.Errorf("Nil.")
		return
	}
	if reflect.ValueOf(v).IsNil() {
		t.Errorf("Nil.")
	}
}

func AssertNil(t *testing.T, v interface{}) {
	t.Helper()
	if v == nil {
		return
	}
	if !reflect.ValueOf(v).IsNil() {
		t.Errorf("Not Nil.")
	}
}

func AssertError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		return
	} else {
		t.Errorf("No Error.")
	}
}

func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	} else {
		t.Errorf("Error.")
	}
}
