// Package httpx contiene helpers compartidos para escribir respuestas JSON
// consistentes y decodificar el cuerpo de las peticiones.
package httpx

import (
	"encoding/json"
	"net/http"
	"reflect"
)

// JSON escribe v como JSON. Antes de codificar, normaliza cualquier slice
// nil (dentro de v o de sus structs anidados) a un slice vacío: en Go, un
// slice sin inicializar se serializa como `null`, pero el frontend siempre
// espera un arreglo (usa .map/.filter sobre la respuesta) y no debería tener
// que distinguir "sin datos todavía" de "ausencia de la lista". Ver
// normalizeForJSON.
func JSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(normalizeForJSON(v))
}

func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}

func Decode(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// normalizeForJSON regresa una versión de v segura para codificar: los
// slices nil de nivel superior, o dentro de structs/punteros anidados,
// quedan como slice vacío en vez de nil. v se pasa casi siempre por valor
// (los servicios regresan structs, no punteros), así que para poder mutar
// campos se construye una copia direccionable cuando hace falta.
func normalizeForJSON(v interface{}) interface{} {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice:
		if rv.IsNil() {
			return reflect.MakeSlice(rv.Type(), 0, 0).Interface()
		}
		for i := 0; i < rv.Len(); i++ {
			normalizeValue(rv.Index(i)) // los elementos de un slice sí son direccionables
		}
		return v
	case reflect.Ptr:
		if !rv.IsNil() {
			normalizeValue(rv.Elem())
		}
		return v
	case reflect.Struct:
		cp := reflect.New(rv.Type()).Elem()
		cp.Set(rv)
		normalizeValue(cp)
		return cp.Interface()
	default:
		return v
	}
}

func normalizeValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.Ptr:
		if !v.IsNil() {
			normalizeValue(v.Elem())
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if !field.CanSet() {
				continue // campo no exportado; no debería llevar datos de respuesta
			}
			switch field.Kind() {
			case reflect.Slice:
				if field.IsNil() {
					field.Set(reflect.MakeSlice(field.Type(), 0, 0))
				} else {
					for j := 0; j < field.Len(); j++ {
						normalizeValue(field.Index(j))
					}
				}
			case reflect.Ptr, reflect.Struct:
				normalizeValue(field)
			}
		}
	}
}
