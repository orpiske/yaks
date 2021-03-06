package cmd

import (
	"fmt"
	"github.com/citrusframework/yaks/pkg/apis/yaks/v1alpha1"
	"github.com/citrusframework/yaks/pkg/cmd/report"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

func newCmdReport(rootCmdOptions *RootCmdOptions) *cobra.Command {
	options := reportCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		PersistentPreRunE: options.preRun,
		Use:               "report [options]",
		Short:             "Generate test report from last test run",
		Long:              `Generate test report from last test run. Test results are fetched from cluster and/or collected from local test output.`,
		RunE:              options.run,
		SilenceUsage:      true,
	}

	cmd.Flags().BoolVar(&options.fetch, "fetch", false, "Fetch latest test results from cluster.")
	cmd.Flags().VarP(&options.output, "output", "o", "The report output format, one of 'summary', 'json', 'junit'")
	cmd.Flags().BoolVarP(&options.clean, "clean", "c", false,"Clean the report output folder before fetching results")

	return &cmd
}

type reportCmdOptions struct {
	*RootCmdOptions
	clean bool
	fetch bool
	output report.OutputFormat
}

func (o *reportCmdOptions) run(cmd *cobra.Command, _ []string) error {
	var results v1alpha1.TestResults
	if o.fetch {
		if fetched, err := o.FetchResults(); err == nil {
			results = *fetched
		} else {
			return err
		}
	} else if loaded, err := report.LoadTestResults(); err == nil {
		results = *loaded
	} else {
		return err
	}

	content, err := report.GenerateReport(&results, o.output)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", content)
	if err != nil {
		return err
	}

	return nil
}

func (o *reportCmdOptions) FetchResults() (*v1alpha1.TestResults, error) {
	c, err := o.GetCmdClient()
	if err != nil {
		return nil, err;
	}

	if o.clean {
		err = report.CleanReports()
		if err != nil {
			return nil, err
		}
	}

	results := v1alpha1.TestResults{}
	testList := v1alpha1.TestList{}
	if err := c.List(o.Context, &testList, ctrl.InNamespace(o.Namespace)); err != nil {
		return nil, err
	}

	for _, test := range testList.Items {
		report.AppendTestResults(&results, test.Status.Results)
		if err := report.SaveTestResults(&test); err != nil {
			fmt.Printf("Failed to save test results: %s", err.Error())
		}
	}

	return &results, nil
}
