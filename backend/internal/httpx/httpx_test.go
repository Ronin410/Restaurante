package httpx

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// Estas pruebas fijan un bug real detectado probando el frontend: Go
// serializa un slice nil como `null`, pero el cliente siempre espera un
// arreglo (usa .map/.filter directo sobre la respuesta) y truena si recibe
// null. JSON() debe entregar siempre `[]`, nunca `null`, para cualquier
// slice dentro de la respuesta.

type innerWithSlice struct {
	Options []string `json:"options"`
}

type outerWithNestedSlices struct {
	Name   string           `json:"name"`
	Items  []int            `json:"items"`
	Inner  innerWithSlice   `json:"inner"`
	Nested []innerWithSlice `json:"nested"`
	Ptr    *innerWithSlice  `json:"ptr"`
}

func decodeBody(t *testing.T, body []byte, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v (body=%s)", err, body)
	}
}

func TestJSON_NilTopLevelSliceBecomesEmptyArray(t *testing.T) {
	var list []int // nil
	rec := httptest.NewRecorder()
	JSON(rec, 200, list)

	if rec.Body.String() != "[]\n" {
		t.Fatalf("esperaba '[]', obtuve %q", rec.Body.String())
	}
}

func TestJSON_NilNestedSliceFieldsBecomeEmptyArrays(t *testing.T) {
	value := outerWithNestedSlices{Name: "x"} // todos los slices quedan nil por defecto
	rec := httptest.NewRecorder()
	JSON(rec, 200, value)

	var raw map[string]interface{}
	decodeBody(t, rec.Body.Bytes(), &raw)

	if _, ok := raw["items"].([]interface{}); !ok {
		t.Fatalf("items debería decodificar como arreglo vacío, obtuve %#v", raw["items"])
	}
	inner := raw["inner"].(map[string]interface{})
	if _, ok := inner["options"].([]interface{}); !ok {
		t.Fatalf("inner.options debería ser arreglo vacío, obtuve %#v", inner["options"])
	}
	if _, ok := raw["nested"].([]interface{}); !ok {
		t.Fatalf("nested debería ser arreglo vacío, obtuve %#v", raw["nested"])
	}
	if raw["ptr"] != nil {
		t.Fatalf("ptr nil debería seguir siendo null, obtuve %#v", raw["ptr"])
	}
}

func TestJSON_NonNilSlicesAreUntouched(t *testing.T) {
	value := outerWithNestedSlices{
		Items: []int{1, 2, 3},
		Inner: innerWithSlice{Options: []string{"a"}},
	}
	rec := httptest.NewRecorder()
	JSON(rec, 200, value)

	var decoded outerWithNestedSlices
	decodeBody(t, rec.Body.Bytes(), &decoded)
	if len(decoded.Items) != 3 || decoded.Items[2] != 3 {
		t.Fatalf("items con datos no debería alterarse, obtuve %#v", decoded.Items)
	}
	if len(decoded.Inner.Options) != 1 || decoded.Inner.Options[0] != "a" {
		t.Fatalf("inner.options con datos no debería alterarse, obtuve %#v", decoded.Inner.Options)
	}
}

func TestJSON_PointerToStructWithNilSlice(t *testing.T) {
	value := &outerWithNestedSlices{Name: "ptr-top"}
	rec := httptest.NewRecorder()
	JSON(rec, 200, value)

	var raw map[string]interface{}
	decodeBody(t, rec.Body.Bytes(), &raw)
	if _, ok := raw["items"].([]interface{}); !ok {
		t.Fatalf("items debería ser arreglo vacío incluso pasando puntero, obtuve %#v", raw["items"])
	}
}
