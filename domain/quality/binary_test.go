package quality

import "testing"

func TestBinaryConcreteBufferEffects(t *testing.T) {
	for _, order := range []string{"littleEndian", "bigEndian"} {
		for _, method := range []string{"PutUint16", "PutUint32", "PutUint64", "AppendUint16", "AppendUint32", "AppendUint64"} {
			name := "encoding/binary." + order + "." + method
			t.Run(name, func(t *testing.T) {
				if !ContainerMutation(name) || ClassifyCallee(name) != MutationEffect {
					t.Fatalf("unmodeled concrete mutation %s", name)
				}
				for _, target := range []Provenance{Local, External, Unknown} {
					f := fact("caller", 1)
					f.Calls = []Call{{Callee: name, Line: 2, MutationTarget: target}}
					got := evaluate(t, f, DefaultConfig()).Functions[0]
					if target == Local {
						if len(got.Effects) != 0 {
							t.Fatal(got)
						}
						continue
					}
					if len(got.Effects) != 1 || got.Effects[0].Kind != MutationEffect {
						t.Fatal(got)
					}
				}
				f := fact("caller", 1)
				f.Calls = []Call{{Callee: name, Line: 2, MutationTarget: Local}}
				config := DefaultConfig()
				config.Prefixes = append(config.Prefixes, Prefix{Path: name, Kind: IO})
				got := evaluate(t, f, config).Functions[0]
				if len(got.Effects) != 1 || got.Effects[0].Kind != IO {
					t.Fatalf("ownership hides explicit IO policy: %+v", got)
				}
			})
		}
		for _, method := range []string{"Uint16", "Uint32", "Uint64"} {
			name := "encoding/binary." + order + "." + method
			if ContainerMutation(name) || ClassifyCallee(name) != None {
				t.Fatalf("read query changes classification: %s", name)
			}
		}
	}
	for _, name := range []string{"encoding/binary.ByteOrder.PutUint16", "encoding/binary.ByteOrder.PutUint32", "encoding/binary.ByteOrder.PutUint64", "encoding/binary.AppendByteOrder.AppendUint16", "encoding/binary.AppendByteOrder.AppendUint32", "encoding/binary.AppendByteOrder.AppendUint64"} {
		if ContainerMutation(name) || ClassifyCallee(name) != UnknownEffect {
			t.Fatalf("interface receives concrete mutation proof: %s", name)
		}
	}
}
