package promote

import (
	"context"
	goerrors "errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/utils/ptr"

	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type promotionOptions struct {
	*option.Option
	Config config.CLIConfig

	FreightName   string
	FreightAlias  string
	Stage         string
	SubscribersOf string
}

func NewCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &promotionOptions{
		Option: opt,
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use: "promote [--project=project] (--freight=freight | --freight-alias=alias) " +
			"(--stage=stage | --subscribers-of=stage)",
		Short: "Promote a piece of freight",
		Args:  option.NoArgs,
		Example: `
# Promote a piece of freight specified by name to the QA stage
kargo promote --project=my-project --freight=abc123 --stage=qa

# Promote a piece of freight specified by alias to the QA stage
kargo promote --project=my-project --freight-alias=wonky-wombat --stage=qa

# Promote a piece of freight specified by name to subscribers of the QA stage
kargo promote --project=my-project --freight=abc123 --subscribers-of=qa

# Promote a piece of freight specified by alias to subscribers of the QA stage
kargo promote --project=my-project --freight-alias=wonky-wombat --subscribers-of=qa

# Promote a piece of freight specified by name to the QA stage in the default project
kargo config set-project my-project
kargo promote --freight=abc123 --stage=qa

# Promote a piece of freight specified by alias to the QA stage in the default project
kargo config set-project my-project
kargo promote --freight-alias=wonky-wombat --stage=qa

# Promote a piece of freight specified by name to subscribers of the QA stage in the default project
kargo config set-project my-project
kargo promote --freight=abc123 --subscribers-of=qa

# Promote a piece of freight specified by alias to subscribers of the QA stage in the default project
kargo config set-project my-project
kargo promote --freight-alias=wonky-wombat --subscribers-of=qas
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdOpts.validate(); err != nil {
				return err
			}

			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	return cmd
}

// addFlags adds the flags for the promotion options to the provided command.
func (o *promotionOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)

	// TODO: Factor out server flags to a higher level (root?) as they are
	//   common to almost all commands.
	option.InsecureTLS(cmd.PersistentFlags(), o.Option)
	option.LocalServer(cmd.PersistentFlags(), o.Option)

	option.Project(
		cmd.Flags(), &o.Project, o.Project,
		"The project the freight belongs to. If not set, the default project will be used.",
	)
	option.Freight(cmd.Flags(), &o.FreightName, "The name of piece of freight to promote.")
	option.FreightAlias(cmd.Flags(), &o.FreightAlias, "The alias of piece of freight to promote.")
	option.Stage(
		cmd.Flags(), &o.Stage,
		fmt.Sprintf(
			"The stage to promote the freight to. If set, --%s must not be set.",
			option.SubscribersOfFlag,
		),
	)
	option.SubscribersOf(
		cmd.Flags(), &o.SubscribersOf,
		fmt.Sprintf(
			"The stage whose subscribers freight should be promoted to. If set, --%s must not be set.",
			option.StageFlag,
		),
	)

	cmd.MarkFlagsOneRequired(option.FreightFlag, option.FreightAliasFlag)
	cmd.MarkFlagsMutuallyExclusive(option.FreightFlag, option.FreightAliasFlag)

	cmd.MarkFlagsOneRequired(option.StageFlag, option.SubscribersOfFlag)
	cmd.MarkFlagsMutuallyExclusive(option.StageFlag, option.SubscribersOfFlag)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *promotionOptions) validate() error {
	var errs []error
	// While the flags are marked as required, a user could still provide an empty
	// string. This is a check to ensure that the flags are not empty.
	if o.Project == "" {
		errs = append(errs, errors.Errorf("%s is required", option.ProjectFlag))
	}
	if o.FreightName == "" && o.FreightAlias == "" {
		errs = append(
			errs,
			errors.Errorf("either %s or %s is required", option.FreightFlag, option.FreightAliasFlag),
		)
	}
	if o.Stage == "" && o.SubscribersOf == "" {
		errs = append(
			errs,
			errors.Errorf("either %s or %s is required", option.StageFlag, option.SubscribersOfFlag),
		)
	}
	return goerrors.Join(errs...)
}

// run performs the promotion of the freight using the options.
func (o *promotionOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.Option)
	if err != nil {
		return err
	}

	switch {
	case o.Stage != "":
		res, err := kargoSvcCli.PromoteStage(
			ctx,
			connect.NewRequest(
				&v1alpha1.PromoteStageRequest{
					Project:      o.Project,
					Freight:      o.FreightName,
					FreightAlias: o.FreightAlias,
					Stage:        o.Stage,
				},
			),
		)
		if err != nil {
			return errors.Wrap(err, "promote stage")
		}
		if ptr.Deref(o.PrintFlags.OutputFormat, "") == "" {
			fmt.Fprintf(o.IOStreams.Out,
				"Promotion Created: %q\n", res.Msg.GetPromotion().GetMetadata().GetName())
			return nil
		}
		printer, err := o.PrintFlags.ToPrinter()
		if err != nil {
			return errors.Wrap(err, "new printer")
		}
		promo := typesv1alpha1.FromPromotionProto(res.Msg.GetPromotion())
		_ = printer.PrintObj(promo, o.IOStreams.Out)
		return nil
	case o.SubscribersOf != "":
		res, promoteErr := kargoSvcCli.PromoteSubscribers(
			ctx,
			connect.NewRequest(
				&v1alpha1.PromoteSubscribersRequest{
					Project:      o.Project,
					Freight:      o.FreightName,
					FreightAlias: o.FreightAlias,
					Stage:        o.SubscribersOf,
				},
			),
		)
		if ptr.Deref(o.PrintFlags.OutputFormat, "") == "" {
			if res != nil && res.Msg != nil {
				for _, p := range res.Msg.GetPromotions() {
					fmt.Fprintf(o.IOStreams.Out, "Promotion Created: %q\n", *p.Metadata.Name)
				}
			}
			if promoteErr != nil {
				return errors.Wrap(promoteErr, "promote subscribers")
			}
			return nil
		}

		printer, printerErr := o.PrintFlags.ToPrinter()
		if printerErr != nil {
			return errors.Wrap(printerErr, "new printer")
		}
		for _, p := range res.Msg.GetPromotions() {
			kubeP := typesv1alpha1.FromPromotionProto(p)
			_ = printer.PrintObj(kubeP, o.IOStreams.Out)
		}
		return promoteErr
	}
	return nil
}
