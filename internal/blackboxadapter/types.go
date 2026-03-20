package blackboxadapter

import bec "blackbox_agent/blackbox_exporter/config"

// ModuleDef is the stable module shape exposed to business code.
// Raw keeps the backend-specific module during the migration period.
type ModuleDef struct {
	Name   string
	Prober string
	Raw    any
}

func moduleDefFromUpstream(name string, module bec.Module) ModuleDef {
	return ModuleDef{
		Name:   name,
		Prober: module.Prober,
		Raw:    module,
	}
}

func upstreamModuleFromDef(module ModuleDef) (bec.Module, bool) {
	rawModule, ok := module.Raw.(bec.Module)
	if !ok {
		return bec.Module{}, false
	}

	return rawModule, true
}
