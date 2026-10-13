package govalidation

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/ccgraph"
	"github.com/rstyled/gstyle/domain/pdg"
)

var primitiveNames = map[types.BasicKind]string{
	types.Bool: "bool", types.Int: "isize", types.Int8: "i8", types.Int16: "i16", types.Int32: "i32", types.Int64: "i64",
	types.Uint: "usize", types.Uint8: "u8", types.Uint16: "u16", types.Uint32: "u32", types.Uint64: "u64", types.Uintptr: "usize",
}

type interfaceValue struct {
	variable *types.Var
	role     pdg.Role
}

func Primitive(typ types.Type) string {
	if typ == nil {
		return ""
	}
	basic, ok := types.Default(typ).Underlying().(*types.Basic)
	if !ok {
		return ""
	}
	return primitiveNames[basic.Kind()]
}

func Interfaces(g *pdg.Graph, fn *ast.FuncDecl, info *types.Info) ([]ccgraph.Parameter, []ccgraph.Parameter, error) {
	object, ok := info.Defs[fn.Name].(*types.Func)
	if !ok {
		return nil, nil, fmt.Errorf("function declaration is unresolved")
	}
	signature, ok := object.Type().(*types.Signature)
	if !ok {
		return nil, nil, fmt.Errorf("function signature is not admitted")
	}
	if !scalarSignature(signature) {
		return nil, nil, fmt.Errorf("generic or variadic signature is not admitted")
	}
	inputs := signatureInputs(signature)
	in, err := interfaceTuples(g, g.Function.Inputs, inputs)
	if err != nil {
		return nil, nil, err
	}
	out, err := interfaceTuples(g, g.Function.Outputs, signatureOutputs(signature))
	return in, out, err
}

func scalarSignature(signature *types.Signature) bool {
	return !signature.Variadic() && signature.TypeParams().Len() == 0 && signature.RecvTypeParams().Len() == 0
}

func signatureInputs(signature *types.Signature) []interfaceValue {
	var values []interfaceValue
	if receiver := signature.Recv(); receiver != nil {
		values = append(values, interfaceValue{receiver, pdg.ReceiverRole})
	}
	for i := 0; i < signature.Params().Len(); i++ {
		values = append(values, interfaceValue{signature.Params().At(i), pdg.ParameterRole})
	}
	return values
}

func signatureOutputs(signature *types.Signature) []interfaceValue {
	var values []interfaceValue
	for i := 0; i < signature.Results().Len(); i++ {
		values = append(values, interfaceValue{signature.Results().At(i), pdg.ReturnRole})
	}
	return values
}

func interfaceTuples(g *pdg.Graph, parameters []pdg.Parameter, expected []interfaceValue) ([]ccgraph.Parameter, error) {
	if len(parameters) != len(expected) {
		return nil, fmt.Errorf("interface cardinality disagrees with source signature")
	}
	out := make([]ccgraph.Parameter, 0, len(parameters))
	for i, parameter := range parameters {
		if err := interfaceElement(g, parameter, expected[i], i); err != nil {
			return nil, err
		}
		role := map[pdg.Role]string{pdg.ParameterRole: "parameter", pdg.ReceiverRole: "receiver", pdg.ReturnRole: "return"}[parameter.Role]
		out = append(out, ccgraph.Parameter{Role: role, Type: Primitive(expected[i].variable.Type()), Variadic: false})
	}
	return out, nil
}

func interfaceElement(g *pdg.Graph, parameter pdg.Parameter, expected interfaceValue, position int) error {
	if parameter.Position != uint32(position) || parameter.Role != expected.role {
		return fmt.Errorf("interface role or position disagrees with signature")
	}
	if !knownScalarParameter(parameter) {
		return fmt.Errorf("interface facts are not complete")
	}
	original := types.TypeString(expected.variable.Type(), func(pkg *types.Package) string { return pkg.Path() })
	if g.Tables.Text(parameter.Type.Value) != original {
		return fmt.Errorf("interface original type disagrees with compiler")
	}
	return resolvedEvidence(g, parameter.Extensions, Primitive(expected.variable.Type()))
}

func knownScalarParameter(parameter pdg.Parameter) bool {
	return parameter.Type.Status == pdg.Known && parameter.Variadic.Status == pdg.Known && !parameter.Variadic.Value
}

func resolvedEvidence(g *pdg.Graph, extensions []pdg.Extension, primitive string) error {
	if primitive == "" {
		return fmt.Errorf("unsupported primitive type")
	}
	count := 0
	for _, extension := range extensions {
		if g.Tables.Text(extension.Name) != "gstyle.go.scalar-normal:resolved-type" {
			continue
		}
		count++
		if err := checkResolvedValue(g.Tables.Text(extension.Value), primitive); err != nil {
			return err
		}
	}
	if count != 1 {
		return fmt.Errorf("resolved type requires exactly one compiler fact")
	}
	return nil
}

func checkResolvedValue(value, primitive string) error {
	var fact struct {
		Primitive string `json:"primitive"`
	}
	if err := json.Unmarshal([]byte(value), &fact); err != nil {
		return err
	}
	if fact.Primitive != primitive {
		return fmt.Errorf("resolved primitive disagrees with compiler")
	}
	return nil
}
