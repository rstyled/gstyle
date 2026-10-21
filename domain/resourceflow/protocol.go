package resourceflow

import (
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

// Protocol identifies typed acquisition and release declarations. These are
// resource contracts, not guarantees about foreign implementations or shutdown.
type Protocol struct {
	Resource string
	Acquire  []string
	Release  []string
}

// Protocols complements the existing file protocol. Keep distinct protocols
// separate: stopping a timer must never discharge a ticker's obligation.
func Protocols() []Protocol {
	return []Protocol{
		{Resource: "ticker", Acquire: []string{"time.NewTicker"}, Release: []string{"time.Ticker.Stop"}},
		{Resource: "timer", Acquire: []string{"time.NewTimer"}, Release: []string{"time.Timer.Stop"}},
		{Resource: "connection", Acquire: []string{"net.Dial", "net.DialTimeout", "net.Dialer.DialContext"}, Release: []string{"net.Conn.Close", "net.TCPConn.Close", "net.UnixConn.Close", "net.UDPConn.Close"}},
		{Resource: "listener", Acquire: []string{"net.Listen"}, Release: []string{"net.Listener.Close", "net.TCPListener.Close", "net.UnixListener.Close"}},
		{Resource: "database", Acquire: []string{"database/sql.Open", "database/sql.OpenDB"}, Release: []string{"database/sql.DB.Close"}},
	}
}

// WithProtocol shares immutable graph indexes. It does not change the original
// query, so package summaries for different resource kinds remain independent.
func (q *Query) WithProtocol(protocol Protocol) *Query {
	if q == nil {
		return nil
	}
	copy := *q
	copy.protocol = protocol
	return &copy
}

func (q *Query) primitiveResource(refName string) bool {
	if q.protocol.Resource == "" {
		return primitiveFileName(refName)
	}
	return slices.Contains(q.protocol.Acquire, refName)
}

// UsesProtocol avoids solving an unrelated protocol over a whole package.
func UsesProtocol(graph *pdg.Graph, protocol Protocol) bool {
	if graph == nil {
		return false
	}
	for _, node := range graph.Nodes {
		if slices.Contains(protocol.Acquire, directResourceCall(graph, node)) {
			return true
		}
	}
	return false
}

func directResourceCall(graph *pdg.Graph, node pdg.Node) string {
	attrs := graph.NodeAttributes(node)
	if node.Category != pdg.Call || attrs.CallForm != pdg.DirectCall || attrs.Callee.Status != pdg.Known {
		return ""
	}
	return graph.Tables.Text(attrs.Callee.Value)
}
