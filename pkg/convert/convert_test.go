package convert

const (
	sampleJSON = `{"name": "Alice", "age": 30}`
	sampleYAML = `
name: Alice
age: 30
`
	sampleTOML = `
name = "Alice"
age = 30
`
	sampleNestedJSON = `{"user": {"name": "Bob", "age": 42}}`
	sampleGoStruct = `
type User struct {
	Name string  ` + "`json:\"name\"`" + `
	Age  int     ` + "`json:\"age\"`" + `
}`
	sampleSchemaJSON = `{"id":1,"active":true,"name":"Test"}`
)
