package enums

/**
 * Andrey Mikhalevich 16/12/21
 * This file is part of the OSBE framework
 *
 * This file is generated from the template build/templates/enum.go.tmpl
 * All direct modification will be lost with the next build.
 * Edit template instead.
 */

import (
	fields "github.com/dronm/crudifier/v2/fields"
)

type ValEnum_locales struct {
	fields.FieldText
}

func (e *ValEnum_locales) GetValues() []string {
	return []string{"ru"}
}

//func (e *ValEnum_locales) GetDescriptions() map[string]map[string]string {
//	return make(map[string]{ "ru" }
//}
