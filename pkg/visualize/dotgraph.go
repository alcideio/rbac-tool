package visualize

import (
	"fmt"
	"html"

	"github.com/emicklei/dot"
)

const (
	fontName = "Poppins 100 normal"

	redOutline = "#e33a1f"

	serviceAccountColor   = "#2f6de1"
	serviceAccountOutline = "black"
	serviceAccountText    = "#f0f0f0"
	serviceAccountShape   = "box"

	roleColor        = "#ff9900"
	roleColorOutline = "black"
	roleColorText    = "#030303"
	roleShape        = "octagon"

	roleBindingColor        = "#ff9900"
	roleBindingColorOutline = "black"
	roleBindingColorText    = "#030303"
	roleBindingShape        = "octagon"

	clusterRoleColor        = "#ffcc00"
	clusterRoleColorOutline = "black"
	clusterRoleColorText    = "#030303"
	clusterRoleShape        = "doubleoctagon"

	clusterRoleBindingColor        = "#ffcc00"
	clusterRoleBindingColorOutline = "black"
	clusterRoleBindingColorText    = "#030303"
	clusterRoleBindingShape        = "doubleoctagon"
)

func newGraph() *dot.Graph {
	g := dot.NewGraph(dot.Directed)

	g.Attr("fontsize", "12.00")
	g.Attr("fontname", fontName)
	// global rank instead of per-subgraph (ensures access rules are always in the same place (at bottom))
	g.Attr("newrank", "true")
	return g
}

func newNamespaceSubgraph(g *dot.Graph, namespace string) *dot.Graph {
	if namespace == "" {
		return g
	}

	gns := g.Subgraph(namespace, dot.ClusterOption{})
	gns.Attr("style", "rounded,dashed")

	return gns
}

func newSubjectNode0(g *dot.Graph, kind, name string, exists, highlight bool) dot.Node {
	return g.Node(kind+"-"+name).
		Box().
		Attr("label", formatLabel(fmt.Sprintf("%s\n(%s)", name, kind), highlight)).
		Attr("style", iff(exists, "filled", "dotted")).
		Attr("color", iff(exists, serviceAccountOutline, redOutline)).
		Attr("penwidth", iff(highlight || !exists, "2.0", "1.0")).
		Attr("margin", "0.22,0.11").
		Attr("fillcolor", serviceAccountColor).
		Attr("fontcolor", iff(exists, serviceAccountText, "#030303")).
		Attr("fontname", fontName)
}

func newRoleBindingNode(g *dot.Graph, name string, highlight bool) dot.Node {
	return g.Node("rb-"+name).
		Attr("label", formatLabel(name, highlight)).
		Attr("shape", roleBindingShape).
		Attr("style", "filled").
		Attr("penwidth", iff(highlight, "2.0", "1.0")).
		Attr("fillcolor", roleBindingColor).
		Attr("color", roleBindingColorOutline).
		Attr("fontcolor", roleBindingColorText).
		Attr("fontname", fontName)
}

func newRoleNode(g *dot.Graph, namespace, name string, exists, highlight bool) dot.Node {
	node := g.Node("r-"+namespace+"/"+name).
		Attr("label", formatLabel(name, highlight)).
		Attr("shape", roleShape).
		Attr("style", iff(exists, "filled", "dotted")).
		Attr("color", iff(exists, roleColorOutline, redOutline)).
		Attr("penwidth", iff(highlight || !exists, "2.0", "1.0")).
		Attr("fillcolor", roleColor).
		Attr("fontcolor", iff(exists, roleColorText, "#030303")).
		Attr("fontname", fontName)
	g.Root().AddToSameRank("Roles", node)
	return node
}

func newClusterRoleBindingNode(g *dot.Graph, name string, highlight bool) dot.Node {
	return g.Node("crb-"+name).
		Attr("label", formatLabel(name, highlight)).
		Attr("shape", roleBindingShape).
		Attr("style", "filled").
		Attr("penwidth", iff(highlight, "2.0", "1.0")).
		Attr("fillcolor", clusterRoleBindingColor).
		Attr("color", clusterRoleBindingColorOutline).
		Attr("fontcolor", clusterRoleBindingColorText).
		Attr("fontname", fontName)
}

func newClusterRoleNode(g *dot.Graph, bindingNamespace, roleName string, exists, highlight bool) dot.Node {
	node := g.Node("cr-"+bindingNamespace+"/"+roleName).
		Attr("label", formatLabel(roleName, highlight)).
		Attr("shape", clusterRoleShape).
		Attr("style", iff(exists, iff(bindingNamespace == "", "filled", "filled,dashed"), "dotted")).
		Attr("color", iff(exists, clusterRoleColorOutline, redOutline)).
		Attr("penwidth", iff(highlight || !exists, "2.0", "1.0")).
		Attr("fillcolor", clusterRoleColor).
		Attr("fontcolor", iff(exists, clusterRoleColorText, "#030303")).
		Attr("fontname", fontName)
	g.Root().AddToSameRank("Roles", node)
	return node
}

func newRulesNode0(g *dot.Graph, namespace, roleName, rulesHTML string, highlight bool) dot.Node {
	return g.Node("rules-"+namespace+"/"+roleName).
		Attr("label", dot.HTML(rulesHTML)).
		Attr("shape", "note").
		Attr("fillcolor", "#DCDCDC").
		Attr("penwidth", iff(highlight, "2.0", "1.0")).
		Attr("fontsize", "10")
}

func formatLabel(label string, highlight bool) interface{} {
	if highlight {
		return dot.HTML("<b>" + html.EscapeString(label) + "</b>")
	} else {
		return label
	}
}

func newSubjectToBindingEdge(subjectNode dot.Node, bindingNode dot.Node) dot.Edge {
	return edge(subjectNode, bindingNode).Attr("dir", "back")
}

func newBindingToRoleEdge(bindingNode dot.Node, roleNode dot.Node) dot.Edge {
	return edge(bindingNode, roleNode)
}

func newRoleToRulesEdge(roleNode dot.Node, rulesNode dot.Node) dot.Edge {
	return edge(roleNode, rulesNode)
}

// edge creates a new edge between two nodes, but only if the edge doesn't exist yet
func edge(from dot.Node, to dot.Node) dot.Edge {
	existingEdges := from.EdgesTo(to)
	if len(existingEdges) == 0 {
		return from.Edge(to)
	} else {
		return existingEdges[0]
	}
}

func iff(condition bool, string1, string2 string) string {
	if condition {
		return string1
	} else {
		return string2
	}
}
