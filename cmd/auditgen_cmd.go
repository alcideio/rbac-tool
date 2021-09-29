package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	auditutil "github.com/alcideio/rbac-tool/pkg/audit"
	"github.com/kylelemons/godebug/pretty"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/apis/audit"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/klog"
)

func NewCommandAuditGen() *cobra.Command {
	options := &AuditGenOpts{
		GeneratedPath:                           ".",
		ExpandMultipleNamespacesToClusterScoped: true,
		ExpandMultipleNamesToUnnamed:            true,
		Annotations: map[string]string{
			"insightcloudsec.rapid7.com/generated-by": "rbac-tool",
			"insightcloudsec.rapid7.com/generated":    time.Now().Format(time.RFC3339),
		},
	}

	cmd := &cobra.Command{
		Use:     "auditgen",
		Aliases: []string{"audit2rbac", "audit", "audit-gen", "a"},
		Short:   "Generate RBAC policy from Kubernetes audit events",
		Long:    "Generate RBAC policy from Kubernetes audit events",
		Example: `

# Generate RBAC policies from audit.log
rbac-tool auditgen -f audit.log 

# Generate RBAC policies fromn audit.log
rbac-tool auditgen -f audit.log -ne '^system:'

# Generate & Visualize 
rbac-tool auditgen -f testdata  | rbac-tool viz   -f -
`,
		RunE: func(cmd *cobra.Command, args []string) error {

			if err := options.Complete(); err != nil {
				return err
			}

			if err := options.Validate(); err != nil {
				return err
			}

			if err := options.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	flags := cmd.Flags()

	flags.StringArrayVarP(&options.AuditSources, "filename", "f", options.AuditSources, "File, Directory, URL, or - for STDIN to read audit events from")
	flags.StringVarP(&options.GeneratedPath, "save", "s", "-", "Save to directory")
	flags.StringVarP(&options.OutputFormat, "output", "o", "yaml", "json or yaml")
	flags.StringVarP(&options.UserRegexFilter, "user", "u", "", "Specify whether run the lookup using a regex match")
	flags.BoolVarP(&options.UserFilterInverse, "not", "n", false, "Inverse the regex matching. Use to search for users that do not match '^system:.*'")

	flags.StringVar(&options.NamespaceRegexFilter, "namespace-filter", ".*", "Namespace regex filter, used to audit events for certain namespaces. By default - all namespaces are evaluated")

	flags.BoolVar(&options.ExpandMultipleNamespacesToClusterScoped, "expand-multi-namespace", options.ExpandMultipleNamespacesToClusterScoped, "Allow identical operations performed in more than one namespace to be performed in any namespace")
	flags.BoolVar(&options.ExpandMultipleNamesToUnnamed, "expand-multi-name", options.ExpandMultipleNamesToUnnamed, "Allow identical operations performed on more than one resource name (e.g. 'get pods pod1' and 'get pods pod2') to be allowed on any name")

	return cmd
}

type AuditGenOpts struct {
	// AuditSources is a list of files, URLs or - for STDIN.
	// Format must be JSON event.v1alpha1.audit.k8s.io, event.v1beta1.audit.k8s.io,  event.v1.audit.k8s.io objects, one per line
	AuditSources []string

	// TODO: Updtate from previously generated policies
	// ExistingObjectFiles is a list of files or URLs.
	// Format must be JSON or YAML RBAC objects or List.v1 objects.
	// ExistingRBACObjectSources []string

	UserRegexFilter   string
	UserFilterInverse bool

	NamespaceRegexFilter string

	// Namespace limits the audit events considered to the specified namespace
	Namespace string

	// JSON or YAML
	OutputFormat string

	// Directory to write generated roles to. Defaults to current directory.
	GeneratedPath string

	// Annotations to apply to generated object names.
	Annotations map[string]string

	// If the same operation is performed in multiple namespaces, expand the permission to allow it in any namespace
	ExpandMultipleNamespacesToClusterScoped bool
	// If the same operation is performed on resources with different names, expand the permission to allow it on any name
	ExpandMultipleNamesToUnnamed bool
}

func (a *AuditGenOpts) Complete() error {
	return nil
}

func (a *AuditGenOpts) Validate() error {
	if len(a.AuditSources) == 0 {
		return fmt.Errorf("--filename is required")
	}

	if len(a.GeneratedPath) == 0 {
		return fmt.Errorf("--output is required")
	}

	if a.UserRegexFilter != "" {
		_, err := regexp.Compile(a.UserRegexFilter)
		if err != nil {
			return fmt.Errorf("--user must be a valid regex - %v", err)
		}
	}

	if a.NamespaceRegexFilter != "" {
		_, err := regexp.Compile(a.NamespaceRegexFilter)
		if err != nil {
			return fmt.Errorf("--namespace-filter must be a valid regex - %v", err)
		}
	}

	return nil
}

func (a *AuditGenOpts) Run() error {
	errs := []error{}

	var userRegex *regexp.Regexp
	var nsRegex *regexp.Regexp
	var err error

	if a.UserRegexFilter != "" {
		userRegex, err = regexp.Compile(a.UserRegexFilter)
	} else {
		userRegex, err = regexp.Compile(fmt.Sprintf(`.*`))
	}

	if err != nil {
		return err
	}

	if a.NamespaceRegexFilter != "" {
		nsRegex, err = regexp.Compile(a.NamespaceRegexFilter)
	} else {
		nsRegex, err = regexp.Compile(fmt.Sprintf(`.*`))
	}

	if err != nil {
		return err
	}

	results, err := auditutil.ReadAuditEvents(a.AuditSources,
		func(event *audit.Event) bool {
			return auditutil.FilterEvent(event, userRegex, a.UserFilterInverse, nsRegex)
		},
	)

	if err != nil {
		errs = []error{err}
	}

	attributesByUser := map[string][]authorizer.AttributesRecord{}
	for result := range results {

		if result.Err != nil {
			errs = append(errs, result.Err)
			klog.V(7).Infof("skipping %v", result.Err)
			continue
		}

		auditEvent := result.Obj.(*audit.Event)
		klog.V(7).Infof("[%v]processing [eventId=%v]", auditEvent.User.Username, auditEvent.AuditID)

		attrs := auditutil.EventToAttributes(auditEvent)

		attributes, exist := attributesByUser[attrs.User.GetName()]
		if !exist {
			attributes = []authorizer.AttributesRecord{}
		}

		attributes = append(attributes, attrs)
		attributesByUser[attrs.User.GetName()] = attributes
	}

	if len(attributesByUser) == 0 {
		message := fmt.Sprintf("No audit events matched user %s", a.UserRegexFilter)
		return fmt.Errorf(message)
	}

	klog.V(7).Infof("processing %v users", len(attributesByUser))

	for username, attributes := range attributesByUser {
		klog.V(7).Infof("[%v] processing %+v", username, pretty.Sprint(username))

		opts := auditutil.DefaultGenerateOptions()
		opts.Annotations = a.Annotations
		opts.Name = fmt.Sprintf("insightcloudsec:%v", sanitizeName(username))
		opts.ExpandMultipleNamespacesToClusterScoped = a.ExpandMultipleNamespacesToClusterScoped
		opts.ExpandMultipleNamesToUnnamed = a.ExpandMultipleNamesToUnnamed

		generated := auditutil.NewGenerator(auditutil.GetDiscoveryRoles(), attributes, opts).Generate()

		f := bufio.NewWriter(os.Stdout)
		defer f.Flush()

		for _, obj := range generated.Roles {
			fmt.Fprintln(f, "\n---\n")
			auditutil.Output(f, obj, a.OutputFormat)
		}
		for _, obj := range generated.ClusterRoles {
			fmt.Fprintln(f, "\n---\n")
			auditutil.Output(f, obj, a.OutputFormat)
		}
		for _, obj := range generated.RoleBindings {
			fmt.Fprintln(f, "\n---\n")
			auditutil.Output(f, obj, a.OutputFormat)
		}
		for _, obj := range generated.ClusterRoleBindings {
			fmt.Fprintln(f, "\n---\n")
			auditutil.Output(f, obj, a.OutputFormat)
		}
	}

	return errors.NewAggregate(errs)
}

func sanitizeName(s string) string {
	return strings.ToLower(string(regexp.MustCompile(`[^a-zA-Z0-9:]`).ReplaceAll([]byte(s), []byte("-"))))
}
