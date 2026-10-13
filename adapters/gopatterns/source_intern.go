package gopatterns

import "github.com/rstyled/gstyle/domain/pdg"

// Source metadata uses the same interned strings as graph facts. The builder's
// construction maps disappear after extraction; source bytes stay file-local.
func internSourceMetadata(pool *pdg.Builder) {
	sources := pool.Tables.Sources
	for i := range sources.Functions {
		sources.Functions[i] = internSourceFunction(pool, sources.Functions[i])
	}
	for i := range sources.References {
		sources.References[i].Target = internSourceText(pool, sources.References[i].Target)
	}
}

func internSourceText(pool *pdg.Builder, value string) string {
	return pool.Tables.Text(pool.Text(value))
}

func internSourceFunction(pool *pdg.Builder, function pdg.SourceFunction) pdg.SourceFunction {
	result := pdg.SourceFunction{ID: internSourceText(pool, function.ID), Name: internSourceText(pool, function.Name), Package: internSourceText(pool, function.Package), Source: function.Source, Declaration: function.Declaration, NameRange: function.NameRange, Parameters: function.Parameters, ParametersList: function.ParametersList, Bindings: function.Bindings, Method: function.Method, Generic: function.Generic, Variadic: function.Variadic}
	for i := range result.Bindings {
		result.Bindings[i] = internSourceText(pool, result.Bindings[i])
	}
	for i := range result.ParametersList {
		result.ParametersList[i] = internSourceParameter(pool, result.ParametersList[i])
	}
	return result
}

func internSourceParameter(pool *pdg.Builder, param pdg.SourceParameter) pdg.SourceParameter {
	return pdg.SourceParameter{Name: internSourceText(pool, param.Name), Type: internSourceText(pool, param.Type), TypeSyntax: internSourceText(pool, param.TypeSyntax), Uses: param.Uses, Addressed: param.Addressed}
}
