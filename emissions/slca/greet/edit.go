package greet

import (
	"fmt"
	"reflect"
)

// EditByID makes changes to db as specified by "changes". Edits are made
// by matching ID numbers in db and changes. Only existing objects can be edited.
// This function has not been exhaustively tested, so use at your own risk.
func (db *DB) EditByID(changes *DB) {
	db.reset()
	editByID(reflect.ValueOf(db), reflect.ValueOf(changes), false)
}

// editByID edits a DB. matchedID determines whether val and changeVal are known
// to have matching types and IDs.
func editByID(val, changeVal reflect.Value, matchedID bool) {
	if !val.IsValid() || !val.CanInterface() ||
		!changeVal.IsValid() || !changeVal.CanInterface() {
		return
	}
	changeType := changeVal.Type()
	switch changeType.Kind() {
	case reflect.Ptr:
		editStruct(val.Elem(), changeVal.Elem(), matchedID)
	case reflect.Struct:
		editStruct(val, changeVal, matchedID)
	case reflect.Array, reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			for j := 0; j < changeVal.Len(); j++ {
				editByID(val.Index(i), changeVal.Index(j), matchedID)
			}
		}
	}
}

// editStruct is called when both val and changeVal are structs.
// It is assumed that they are of the same type.
// The return value indicates whether val and changeVal have matching
// IDs and types.
func editStruct(val, changeVal reflect.Value, matchedID bool) {
	if !changeVal.IsValid() {
		return
	}
	if matchedID {
		if changeVal.Type().Name() == "ValueYear" {
			editValueYear(val, changeVal)
		} else if changeVal.Type().Name() == "OtherEmission" ||
			changeVal.Type().Name() == "MixPathway" {
			// OtherEmissions are matched by pollutant ref rather than ID.
			changeRef := changeVal.FieldByName("Ref")
			valRef := val.FieldByName("Ref")
			if changeRef.String() == valRef.String() {
				// The IDs match. Now we edit.
				directlyEditStruct(val, changeVal)
			}
			return
		}
	}
	changeID := changeVal.FieldByName("ID")
	if !changeID.IsValid() {
		// changeVal doesn't contain actual changes because it doesn't
		// have an ID, so we check if any of its fields contain changes
		// that need to be made.
		for i := 0; i < changeVal.NumField(); i++ {
			editByID(val.Field(i), changeVal.Field(i), matchedID)
		}
		return
	}
	// changeVal does contain an ID, so we know it has changes that
	// need to be made.
	valID := val.FieldByName("ID")
	if changeID.String() == valID.String() {
		// The IDs match. Now we edit.
		directlyEditStruct(val, changeVal)
	}
}

// directlyEditStruct edits a struct. This and editValueYear are the only
// places where we actually make any changes to val.
func directlyEditStruct(val, changeVal reflect.Value) {
	for i := 0; i < changeVal.NumField(); i++ {
		changeField := changeVal.Field(i)
		valField := val.Field(i)
		fieldType := changeField.Type()
		zero := reflect.Zero(fieldType)
		if fieldType.Kind() == reflect.String {
			// We only directly edit string fields.
			if changeField != zero && changeField.String() != "" {
				valField.Set(changeField)
			}
		} else {
			// Since this is not a string field, it may be a struct or array that
			// contains additional information to be edited. Since we have just
			// found matchedIDs, set matchedID to true.
			editByID(valField, changeField, true)
		}
	}
}

// editValueYear checks whether the "Year" fields match in val and changeVal
// and if they do it replaces the "Value" field in val with the changeVal value.
func editValueYear(val, changeVal reflect.Value) {
	valYear := val.FieldByName("Year")
	changeYear := changeVal.FieldByName("Year")
	if valYear.String() == changeYear.String() {
		val.FieldByName("Value").Set(changeVal.FieldByName("Value"))
	}
}

// EditExpressionByID finds the expression with an ID matching "ID" and
// sets the value to newValue. It returns an error if "ID" is not found or
// if there is more than one expression matching "ID" in the database.
func (db *DB) EditExpressionByID(ID string, newValue float64) error {
	db.reset()
	found, err := editExpressionByID(reflect.ValueOf(db), ID, newValue, false)
	if !found {
		err = fmt.Errorf("could not find expression %s", ID)
	}
	return err
}

// editExpressionByID edits a DB. matchedID determines whether val and changeVal are known
// to have matching types and IDs.
func editExpressionByID(val reflect.Value, ID string, newValue float64, found bool) (bool, error) {
	var err error
	if !val.IsValid() || !val.CanInterface() {
		return found, err
	}
	valType := val.Type()
	switch valType.Kind() {
	case reflect.Ptr:
		found, err = editExpressionByID(val.Elem(), ID, newValue, found)
		if err != nil {
			return found, err
		}
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			found, err = editExpressionByID(val.Field(i), ID, newValue, found)
			if err != nil {
				return found, err
			}
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			found, err = editExpressionByID(val.Index(i), ID, newValue, found)
			if err != nil {
				return found, err
			}
		}
	case reflect.String:
		if val.Type().Name() == "Expression" {
			e := val.Interface().(Expression).decode()
			for _, eID := range e.names {
				if eID == ID {
					if found {
						return found, fmt.Errorf("multiple matches for expression %s", ID)
					}
					e.val = fmt.Sprintf("%g", newValue)
					val.SetString(string(e.encode()))
					found = true
				}
			}
		}
	}
	return found, nil
}
