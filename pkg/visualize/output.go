package visualize

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/alcideio/rbac-tool/pkg/utils"
	"github.com/emicklei/dot"
	"k8s.io/klog"
	"text/template"
)

func GenerateOutput(filename string, format string, g *dot.Graph, legend *dot.Graph, opts *Opts) error {

	switch format {

	case "html":
		report := HtmlReport{
			Graph:  g,
			Legend: legend,
			opts:   opts,
		}

		out, err := report.Generate()
		if err != nil {
			return err
		}

		return utils.WriteFile(filename, out)
	case "dot":
		fallthrough
	default:
		return utils.WriteFile(filename, g.String())
	}

}

type HtmlReport struct {
	Graph   *dot.Graph
	Legend  *dot.Graph
	opts    *Opts
	counter int64
}

func (r *HtmlReport) generateHeader() string {

	data := `
		<nav class="navbar navbar-expand-lg navbar-dark bg-info mb-4">
			<div class="row">
				<div class="col-6 collapse-brand">
					<a href="javascript:void(0)">
						<img src="https://raw.githubusercontent.com/alcideio/rbac-tool/master/rbac-tool.png" height="48">
					</a>
				</div>
				<div class="col-6 collapse-close">
					<button type="button" class="navbar-toggler" data-toggle="collapse" data-target="#navbar-default" aria-controls="navbar-default" aria-expanded="false" aria-label="Toggle navigation">
						<span></span>
						<span></span>
					</button>
				</div>
			</div>
			<div class="container">
				<a class="navbar-brand" href="#"><span style="font-weight: 100; font-style: normal; font-size: 22px;">Alcide | RBAC Tool</span></a>
				<button class="navbar-toggler" type="button" data-toggle="collapse" data-target="#navbar-default" aria-controls="navbar-default" aria-expanded="false" aria-label="Toggle navigation">
					<span class="navbar-toggler-icon"></span>
				</button>
				<div class="collapse navbar-collapse" id="navbar-default">
					<div class="navbar-collapse-header">
						<div class="row">
							<div class="col-6 collapse-brand">
							</div>
							<div class="col-6 collapse-close">
								<button type="button" class="navbar-toggler" data-toggle="collapse" data-target="#navbar-default" aria-controls="navbar-default" aria-expanded="false" aria-label="Toggle navigation">
									<span></span>
									<span></span>
								</button>
							</div>
						</div>
					</div>

					<ul class="navbar-nav ml-auto">
						<li class="nav-item">
							<a class="nav-link nav-link-icon" href="https://github.com/alcideio/rbac-tool" target="_blank">
								<i class="fab fa-github"></i>
								<span class="nav-link-inner--text">GitHub</span>
							</a>
						</li>
						<li class="nav-item">
						  <a class="nav-link nav-link-icon" href="https://www.alcide.io/" target="_blank">
							<i class="fas fa-home"></i>
							<span class="nav-link-inner--text">Site</span>
						  </a>
						</li>
						<li class="nav-item">
							<a class="nav-link nav-link-icon" href="https://codelab.alcide.io" target="_blank">
								<i class="fas fa-code"></i>
								<span class="nav-link-inner--text">Codelabs</span>
							</a>
						</li>
						<li class="nav-item">
						  <a class="nav-link nav-link-icon" href="https://twitter.com/alcideio" target="_blank">
							<i class="fab fa-twitter"></i>
							<span class="nav-link-inner--text">Twitter</span>
						  </a>
						</li>
				  </ul>
				</div>
			</div>
		</nav>
`

	tmpl, err := r.newTemplateEngine("header", data)
	if err != nil {
		klog.Errorf("Failed to generate category chart - %v", err)
		return ""
	}

	buf := new(bytes.Buffer)

	err = tmpl.Execute(buf, r)
	if err != nil {
		klog.Errorf("Failed to execute category chart - %v", err)
		return ""
	}

	return buf.String()
}

func (r *HtmlReport) generateFooter() string {

	data := `
		<div class="row p20 mt-4">
					<div class="col-md-4 text-center mt-4">
						</div>
					<div class="col-md-4 text-center p10">
						<div>
 							<span>
								<a href="javascript:void(0)"><img src="https://raw.githubusercontent.com/alcideio/rbac-tool/master/rbac-tool.png" height="48"></a>
								<p style="font-weight: 100; font-style: normal; font-size: 18px;">Brought to You by <a target="_blank" href="https://www.alcide.io">Alcide's</a> Kubernetes Obsession</p>
							</span>
						</div>
					</div>
					<div class="col-md-4 text-center">
					</div>
		</div>
`

	tmpl, err := r.newTemplateEngine("footer", data)
	if err != nil {
		klog.Errorf("Failed to generate footer - %v", err)
		return ""
	}

	buf := new(bytes.Buffer)

	err = tmpl.Execute(buf, r)
	if err != nil {
		klog.Errorf("Failed to execute footer - %v", err)
		return ""
	}

	return buf.String()
}

func (r *HtmlReport) generateBody() string {
	data := `
		<div class="container-fluid">
			<div class="row">
				<div class="col-3">
					<div class="card border-light"">

					  <ul class="list-group list-group-flush">
						<div><li class="list-group-item border-ligh">{{ generateGraph .Legend "legend" "100%" }}</li></div>
					  </ul>
					  <div class="card-header text-center">
						Legend
					  </div>
					</div>
				</div>
				<div class="col-6">
					{{ generateGraph .Graph "rbacgraph" "auto" }}
				</div>
				<div class="col-3">
				</div>
			</div>
		</div>
`
	funcs := sprig.TxtFuncMap()

	funcs["uniqueCounter"] = r.uniqueCounter
	funcs["generateGraph"] = r.generateGraph

	tmpl, err := r.newTemplateEngine("body", data)
	if err != nil {
		klog.Errorf("Failed to generate body - %v", err)
		return ""
	}

	buf := new(bytes.Buffer)

	err = tmpl.Funcs(funcs).Execute(buf, r)
	if err != nil {
		klog.Errorf("Failed to execute category chart - %v", err)
		return ""
	}

	return buf.String()

}

func (r *HtmlReport) generateGraph(graph *dot.Graph, divId string, widthStyle string) string {
	fmtHtmlCode := `
<style>
  #` + divId + ` svg {
    height: auto;
    width: ` + widthStyle + `;
  }
</style>
<div id="` + divId + `" style="text-align: center;">
</div>

<script>
var dotSrc` + divId + ` = ` + fmt.Sprintf("`%s`", graph.String()) + `;

var graphviz` + divId + ` = d3.select("#` + divId + `").graphviz()
    .transition(function () {
        return d3.transition("main")
            .ease(d3.easeLinear)
            .delay(500)
            .duration(1500);
    })
    .logEvents(true)
    .on("initEnd", render` + divId + `);

function render` + divId + `() {
    console.log('DOT source =', dotSrc` + divId + `);
    dotSrcLines = dotSrc` + divId + `.split('\n');

    graphviz` + divId + `
        .transition(function() {
            return d3.transition()
                .delay(100)
                .duration(1000);
        })
        .renderDot(dotSrc` + divId + `)
		.zoom(true);
}

</script>
`

	return fmtHtmlCode
}

func (r *HtmlReport) uniqueCounter() string {
	r.counter++

	return fmt.Sprint(r.counter)
}

func (r *HtmlReport) newTemplateEngine(name string, data string) (*template.Template, error) {
	funcs := sprig.TxtFuncMap()

	funcs["uniqueCounter"] = r.uniqueCounter
	funcs["generateHeader"] = r.generateHeader
	funcs["generateBody"] = r.generateBody
	funcs["generateFooter"] = r.generateFooter
	funcs["generateGraph"] = r.generateGraph

	return template.New(name).Funcs(funcs).Parse(data)
}

func (r *HtmlReport) Generate() (out string, err error) {

	html := `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
  <link rel="shortcut icon" href="https://www.alcide.io/wp-content/themes/alcide/favicon.ico" />
  <title>[Alcide] RBAC Tool for Kubernetes</title>

  <!-- Fonts -->
  <link href="https://fonts.googleapis.com/css?family=Poppins:200,300,400,600,700,800" rel="stylesheet">

  <!-- Icons -->
  <link href="https://use.fontawesome.com/releases/v5.0.6/css/all.css" rel="stylesheet">

  <!-- Argon CSS -->
  <link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.4.1/css/bootstrap.min.css" integrity="sha384-Vkoo8x4CGsO3+Hhxv8T/Q5PaXtkKtu6ug5TOeNV6gBiFeWPGFN9MuhOf23Q9Ifjh" crossorigin="anonymous">
</head>
	<body>
		<script src="https://d3js.org/d3.v5.min.js"></script>
		<script src="https://unpkg.com/@hpcc-js/wasm/dist/index.min.js"></script>
		<script src="https://unpkg.com/d3-graphviz@3.0.0/build/d3-graphviz.js"></script>
		
		<!-- Core -->
		<script src="https://code.jquery.com/jquery-3.4.1.slim.min.js" integrity="sha384-J6qa4849blE2+poT4WnyKhv5vZF5SrPo0iEjwBvKU7imGFAV0wwj1yYfoRSJoZ+n" crossorigin="anonymous"></script>
		<script src="https://cdn.jsdelivr.net/npm/popper.js@1.16.0/dist/umd/popper.min.js" integrity="sha384-Q6E9RHvbIyZFJoft+2mJbHaEWldlvI9IOYy5n3zV9zzTtmI3UksdQRVvoxMfooAo" crossorigin="anonymous"></script>
		<script src="https://stackpath.bootstrapcdn.com/bootstrap/4.4.1/js/bootstrap.min.js" integrity="sha384-wfSDF2E50Y2D1uUdj0O3uMBJnjuUD4Ih7YwaYd1iqfktj0Uod8GCExl3Og8ifwB6" crossorigin="anonymous"></script>
		<!-- script src="/assets/js/argon-design-system.min.js"></script -->

		{{ generateHeader }}

		{{ generateBody }}

		{{ generateFooter }}

	</body>
</html>
  `

	tmpl, err := r.newTemplateEngine("full-report", html)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)

	err = tmpl.Execute(buf, r)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
