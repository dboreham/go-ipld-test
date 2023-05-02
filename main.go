package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/dboreham/go-ipld-test/model"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagjson"
	"github.com/ipld/go-ipld-prime/node/bindnode"

	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func main() {

	ts, err := ipld.LoadSchemaBytes([]byte(`
	type Person struct {
		Name    String
		Age     optional Int
		Friends optional [String]
	}
`))
	if err != nil {
		panic(err)
	}
	schemaType := ts.TypeByName("Person")

	type Person struct {
		Name    string
		Age     *int64   // optional
		Friends []string // optional; no need for a pointer as slices are nilable
	}
	ipldPerson := &Person{
		Name:    "Michael",
		Friends: []string{"Sarah", "Alex"},
	}
	personNode := bindnode.Wrap(ipldPerson, schemaType)

	personNodeRepr := personNode.Representation()
	dagjson.Encode(personNodeRepr, os.Stdout)

	// Demonstrate the generated Go protobuf code using:
	// ```
	// go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	// protoc --go_out=. --go_opt=paths=source_relative *.proto
	// ```

	// Create an instance using generated go type for message Person
	person := &model.Person{
		Name: "Alex",
		Age:  20,
	}
	fmt.Println("Original person: ", person)

	// Serialize the instance into byte array
	personData, err := proto.Marshal(person)
	if err != nil {
		panic(err)
	}

	// Deserialze the bytes into a new instance
	newPerson := &model.Person{}
	err = proto.Unmarshal(personData, newPerson)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Deserialized person: %v\n", newPerson)

	// Demonstrate FileDescriptor using; the descriptor set is generated using:
	// ```
	// protoc --proto_path=. --include_imports --descriptor_set_out=descriptor.pb person.proto
	// ```

	// Extract the MessageDescriptor required to create a DynamicMessage from the generated descriptor set
	sourceDescriptorFilePath := "model/descriptor.pb"
	protoFilePath := "person.proto"
	protoMessageName := protoreflect.Name("Person")

	// Read the descriptor file into memory
	descriptorSet, err := ioutil.ReadFile(sourceDescriptorFilePath)
	if err != nil {
		log.Fatalf("Error reading .proto file: %v", err)
	}

	// Parse the descriptor set into a FileDescriptorSet
	fileDescriptorSet := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(descriptorSet, fileDescriptorSet); err != nil {
		log.Fatalf("Error parsing FileDescriptorSet: %v", err)
	}

	// Convert FileDescriptorProto messages from the set into 'protoreflect.FileDescriptor's
	protoRegistryFiles, err := protodesc.NewFiles(fileDescriptorSet)
	if err != nil {
		log.Fatalf("Error creating a protoRegistry from a fileDescriptorSet: %v", err)
	}

	// Get the required FileDescriptor
	fileDescriptor, err := protoRegistryFiles.FindFileByPath(protoFilePath)
	if err != nil {
		log.Fatalf("Error finding descriptor for file %s: %v", protoFilePath, err)
	}

	// Print out the FileDescriptor
	fmt.Println()
	fmt.Println("FileDescriptor: ", fileDescriptor)
	fmt.Println()

	// Print out the path of the file and the package name
	fmt.Println("File path: ", fileDescriptor.Path())
	fmt.Println("Package name: ", fileDescriptor.Package())
	fmt.Println("Messages: ", fileDescriptor.Messages())
	fmt.Println()

	// Get the required MessageDescriptor
	personMessageDescriptor := fileDescriptor.Messages().ByName(protoMessageName)
	fmt.Println("Person MessageDescriptor:", personMessageDescriptor)
	fmt.Println()

	// Create and populate dynamicpb.Message instance from the extracted MessageDescriptor
	dynamicMessage := dynamicpb.NewMessage(personMessageDescriptor)
	if err := proto.Unmarshal(personData, dynamicMessage); err != nil {
		log.Fatalf("Error parsing personData: %v", err)
	}
	fmt.Printf("Deserialized dynamic person: %v\n", dynamicMessage)

	// Using go-ipld-prime bindnode to IPLDize the created dynamic message

	// Use without schema and let go-ipld-prime try to infer it using reflection
	// https://github.com/ipld/go-ipld-prime/blob/master/node/bindnode/example_test.go#L46
	// This fails as dynamicMessage.typ.desc is an interface
	node := bindnode.Wrap(dynamicMessage, nil)

	nodeRepr := node.Representation()
	dagjson.Encode(nodeRepr, os.Stdout)

	// Code that does the same thing internally
	// encoded, err := ipld.Marshal(dagjson.Encode, dynamicMessage, nil)
	// if err != nil {
	// 	panic(err)
	// }
}
