package visualize

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/alcideio/rbac-tool/pkg/utils"
	"github.com/emicklei/dot"
)

type HtmlReport struct {
	Graph   *dot.Graph
	Legend  *dot.Graph
	opts    *Opts
	counter int64
}

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

func (r *HtmlReport) Generate() (out string, err error) {
	html := `
<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="shortcut icon" href="https://www.rapid7.com/includes/img/favicon.ico" />
    <title>[Rapid7 | InsightCloudSec] Kubernetes RBAC Power Toys</title>

    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/bootstrap-icons.min.css"
        integrity="sha384-XGjxtQfXaH2tnPFa9x+ruJTuLE3Aa6LhHSWRr1XeTyhezb4abCG4ccI5AkVDxqC+" crossorigin="anonymous">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css"
        integrity="sha384-QWTKZyjpPEjISv5WaRU9OFeRpok6YctnYmDr5pNlyT2bRjXh0JMhjY6hW+ALEwIH" crossorigin="anonymous">

    <style>
        body {
            overflow-x: hidden;
        }

        .nav-title {
            font-size: 22px;
            font-weight: 100;
            font-style: normal;
        }

        .popover {
            max-width: 100% !important;
        }

        #rbacgraph svg {
            height: calc(100vh - 110px);
            width: auto;
        }

        #legend svg {
            height: auto;
            width: 100%;
            max-width: 500px;
        }
    </style>
</head>

<body>
    <nav class="navbar navbar-expand-sm bg-dark" data-bs-theme="dark">
        <div class="container-fluid">
            <a class="navbar-brand" href="https://www.rapid7.com/products/insightcloudsec">
                <img src="https://www.rapid7.com/includes/img/Rapid7_logo.svg" height="24">
                <img src="https://www.rapid7.com/globalassets/_logos/insightcloudsec-w.svg" height="28">
                <span class="nav-title"> | RBAC Tool</span>
            </a>
            <div class="d-flex justify-content-end">
                <div class="navbar-nav">
                    <a class="nav-link" href="https://twitter.com/rapid7">
                        <i class="bi-twitter"></i>
                    </a>
                    <a class="nav-link" href="https://github.com/alcideio/rbac-tool">
                        <i class="bi-github"></i>
                    </a>
                </div>
            </div>
        </div>
    </nav>

    <div class="container-fluid">
        <div class="row">
            <div class="col">
                <button id="button-legend" class="btn btn-secondary position-fixed m-3" type="button"
                    data-bs-container="body" data-bs-toggle="popover" data-bs-placement="bottom">
                    <i class="bi-grid"></i> Legend
                </button>

                <div class="container-legend" hidden>
                    <div id="legend" class="text-center" data-name="popover-content"></div>
                </div>

                <div id="rbac" class="text-center"></div>

                <p class="text-center fixed-bottom">
                    Brought to You by <a target="_blank" href="https://www.rapid7.com/products/insightcloudsec/">Rapid7 InsightCloudSec</a> Kubernetes Obsession
                </p>
            </div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/d3@5.16.0/dist/d3.min.js"
        integrity="sha256-Xb6SSzhH3wEPC4Vy3W70Lqh9Y3Du/3KxPqI2JHQSpTw=" crossorigin="anonymous"></script>

    <script src="https://unpkg.com/viz.js@1.8.1/viz.js"
        integrity="sha256-ceV+JPvxGEp8n+NFiNhtOgVQ8w8ASMjmpAwz3NpW86U=" crossorigin="anonymous"></script>

    <script src="https://cdn.jsdelivr.net/npm/d3-graphviz@2.6.1/build/d3-graphviz.min.js"
        integrity="sha256-ga8Whvn2wFe328K4mwyb9iQQFWOtxO2nwg7Ey8lLido=" crossorigin="anonymous"></script>

    <script src="https://cdn.jsdelivr.net/npm/@popperjs/core@2.11.8/dist/umd/popper.min.js"
        integrity="sha256-whL0tQWoY1Ku1iskqPFvmZ+CHsvmRWx/PIoEvIeWh4I=" crossorigin="anonymous"></script>

    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js"
        integrity="sha384-YvpcrYf0tY3lHB60NNkmXc5s9fDVZLESaAA55NDzOxhy9GkcIdslK1eN7N6jIeHz"
        crossorigin="anonymous"></script>

    <script>
        const options = { html: true, content: document.getElementById("legend") }
        new bootstrap.Popover(document.getElementById("button-legend"), options)

		{{ generateGraph .Legend "legend" }}
		{{ generateGraph .Graph "rbac" }}
    </script>
</body>

</html>`

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

func (r *HtmlReport) generateGraph(graph *dot.Graph, divId string) string {
	fmtHtmlCode := `
const dotSource` + divId + ` = ` + fmt.Sprintf("`%s`", graph.String()) + `;

const graphviz` + divId + ` = d3.select("#` + divId + `").graphviz()
	.logEvents(true).on("initEnd", () =>
		graphviz` + divId + `.renderDot(dotSource` + divId + `).zoom(true)
	);`

	return fmtHtmlCode
}

func (r *HtmlReport) uniqueCounter() string {
	r.counter++
	return fmt.Sprint(r.counter)
}

func (r *HtmlReport) newTemplateEngine(name string, data string) (*template.Template, error) {
	funcs := sprig.TxtFuncMap()
	funcs["uniqueCounter"] = r.uniqueCounter
	funcs["generateGraph"] = r.generateGraph

	return template.New(name).Funcs(funcs).Parse(data)
}
