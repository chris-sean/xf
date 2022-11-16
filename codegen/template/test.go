package template

const TEST_GO = `// #Annotation#
package test

import (
	"encoding/json"
	"fmt"
	"testing"
)

const #TypeName#AddJson = ` + "`" + `
{

}
` + "`" + `

const #TypeName#UpdateJson = ` + "`" + `
{
"id": %v
}
` + "`" + `

func Test#TypeName#(t *testing.T) {
	var id any

	t.Cleanup(func() {
		testDelete#TypeName#(t, id)
		testGet#TypeName#AfterDeletion(t, id)
	})

	id = testAdd#TypeName#(t)
	testGet#TypeName#(t, id)
	testUpdate#TypeName#(t, id)
	testGet#TypeName#Page(t, id, true)
}

func testAdd#TypeName#(t *testing.T) any {
	api := "#type-name#/add"

	log.Printf("%v test %v", timestamp(), api)

	req := #TypeName#AddJson

	resp, err := client.R().
		SetBodyJsonString(req).
		Post(api)

	if err != nil {
		t.Fatal(err)
	}

	if resp.IsError() {
		t.Fatal(resp.StatusCode, resp.Proto, resp.Header)
	}

	ro := &struct {
		CommonResponseJSON
		ID any ` + "`" + `json:"id"` + "`" + `
	}{}

	err = resp.Unmarshal(ro)

	if err != nil {
		t.Fatal(err)
	}

	if ro.Error != nil {
		t.Fatal(ro.Error)
	}

	switch ro.ID.(type) {
	case float64:
		id := int64(ro.ID.(float64))
		if id <= 0 {
			t.Fatalf("#type_name# id can not be %v", ro.ID)
		}
	case int64:
		id := ro.ID.(int64)
		if id <= 0 {
			t.Fatalf("#type_name# id can not be %v", ro.ID)
		}
	case string:
		id := ro.ID.(string)
		if id == "" {
			t.Fatalf("#type_name# id can not be %v", ro.ID)
		}

		return fmt.Sprintf("\"%v\"", id)
	}

	return ro.ID
}

func testUpdate#TypeName#(t *testing.T, id any) {
	api := "#type-name#/set"

	log.Printf("%v test %v", timestamp(), api)

	req := fmt.Sprintf(#TypeName#UpdateJson, id)

	resp, err := client.R().
		SetBodyJsonString(req).
		Post(api)

	if err != nil {
		t.Fatal(err)
	}

	if resp.IsError() {
		t.Fatal(resp.StatusCode, resp.Proto, resp.Header)
	}

	ro := &struct {
		CommonResponseJSON
	}{}

	err = resp.Unmarshal(ro)

	if err != nil {
		t.Fatal(err)
	}

	if ro.Error != nil {
		t.Fatal(ro.Error)
	}
}

func get#TypeName#(t *testing.T, id any) map[string]any {
	api := "#type-name#/get"

	log.Printf("%v call %v", timestamp(), api)

	req := fmt.Sprintf(` + "`" + `
{
"id": %v
}
` + "`" + `, id)

	resp, err := client.R().
		SetBodyJsonString(req).
		Post(api)

	if err != nil {
		t.Fatal(err)
	}

	if resp.IsError() {
		t.Fatal(resp.StatusCode, resp.Proto, resp.Header)
	}

	ro := &struct {
		CommonResponseJSON
		Data map[string]any ` + "`" + `json:"#type_name#"` + "`" + `
	}{}

	err = resp.Unmarshal(ro)

	if err != nil {
		t.Fatal(err)
	}

	if ro.Error != nil {
		t.Fatal(ro.Error)
	}

	return ro.Data
}

func testGet#TypeName#(t *testing.T, id any) {
	log.Printf("%v test get #type_name#", timestamp())

	data := get#TypeName#(t, id)

	var addedData map[string]any
	if err := json.Unmarshal([]byte(#TypeName#AddJson), &addedData); err != nil {
		t.Fatal(err)
	}

	testRightContainsLeft(t, addedData, data)
}

func testGet#TypeName#Page(t *testing.T, id any, pagination bool) {
	api := "#type-name#/page"

	log.Printf("%v test %v", timestamp(), api)

	var req string

	if pagination {
		req = fmt.Sprintf(` + "`" + `
{
"page": 1,
"size": 10,
"match": {
"id": %v
}
}
` + "`" + `, id)
	} else {
		req = fmt.Sprintf(` + "`" + `
{
"id": %v
}
` + "`" + `, id)
	}

	resp, err := client.R().
		SetBodyJsonString(req).
		Post(api)

	if err != nil {
		t.Fatal(err)
	}

	if resp.IsError() {
		t.Fatal(resp.StatusCode, resp.Proto, resp.Header)
	}

	ro := &struct {
		CommonResponseJSON
		Data map[string]any ` + "`" + `json:"#list_name#"` + "`" + `
	}{}

	err = resp.Unmarshal(ro)

	if err != nil {
		t.Fatal(err)
	}

	if ro.Error != nil {
		t.Fatal(ro.Error)
	}

	var e map[string]any

    if pagination {
        data, ok := ro.Data["data"].([]any)

        if !ok {
            t.Fatalf("expect data to be an array. got data=%#v", ro.Data["data"])
        }

        if len(data) != 1 {
            t.Fatalf("len(data)=%v, expect=1", len(data))
        }

        e, ok = data[0].(map[string]any)

        if !ok {
            t.Fatalf("expect type of map[string]any. got data[0]=%#v", data[0])
        }
    } else {
        e = ro.Data
    }

	var updatedData map[string]any
	if err = json.Unmarshal([]byte(fmt.Sprintf(#TypeName#UpdateJson, id)), &updatedData); err != nil {
		t.Fatal(err)
	}

	testRightContainsLeft(t, updatedData, e)
}

func testDelete#TypeName#(t *testing.T, id any) {
	switch id.(type) {
	case float64:
		i := int64(id.(float64))
		if i <= 0 {
			return
		}
	case int64:
		i := id.(int64)
		if i <= 0 {
			return
		}
	case string:
		i := id.(string)
		if i == "" {
			return
		}
	}

	api := "#type-name#/del"

	log.Printf("%v test %v", timestamp(), api)

	req := fmt.Sprintf(` + "`" + `
{
"id": %v
}
` + "`" + `, id)

	resp, err := client.R().
		SetBodyJsonString(req).
		Post(api)

	if err != nil {
		t.Fatal(err)
	}

	if resp.IsError() {
		t.Fatal(resp.StatusCode, resp.Proto, resp.Header)
	}

	ro := &struct {
		CommonResponseJSON
	}{}

	err = resp.Unmarshal(ro)

	if err != nil {
		t.Fatal(err)
	}

	if ro.Error != nil {
		t.Fatal(ro.Error)
	}
}

func testGet#TypeName#AfterDeletion(t *testing.T, id any) {
	log.Printf("%v test get #type_name# after deletion", timestamp())

	data := get#TypeName#(t, id)

	if data != nil {
		t.Fatalf("expect=nil, got=%v;", data)
	}
}
`
