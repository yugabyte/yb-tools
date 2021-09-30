package flag

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func BindFlags(flags *pflag.FlagSet) {
	replacer := strings.NewReplacer("-", "_")

	flags.VisitAll(func(flag *pflag.Flag) {
		if err := viper.BindPFlag(replacer.Replace(flag.Name), flag); err != nil {
			panic("unable to bind flag " + flag.Name + ": " + err.Error())
		}
	})
}

func ValidateRequiredFlags(flags *pflag.FlagSet) error {
	var required []*pflag.Flag
	flags.VisitAll(func(flag *pflag.Flag) {
		if len(flag.Annotations[YBToolsMarkFlagRequired]) > 0 {
			required = append(required, flag)
		}
	})

	var missing []string
	for _, flag := range required {
		replacer := strings.NewReplacer("-", "_")
		vname := replacer.Replace(flag.Name)
		// The flag was specified in the command line
		if flag.Changed {
			continue
		}

		value := viper.Get(vname)
		if value != nil {
			if !reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface()) {
				continue
			}
		}

		missing = append(missing, flag.Name)
	}

	if len(missing) > 0 {
		return fmt.Errorf("required flag(s) %v not set", missing)
	}
	return nil
}

const (
	YBToolsMarkFlagRequired = "yugaware_client_flag_required_annotation"
)

func MarkFlagRequired(name string, flags *pflag.FlagSet) {
	err := flags.SetAnnotation(name, YBToolsMarkFlagRequired, []string{"true"})
	if err != nil {
		panic("could not mark flag required: " + err.Error())
	}
}

func MarkFlagsRequired(names []string, flags *pflag.FlagSet) {
	for _, name := range names {
		MarkFlagRequired(name, flags)
	}
}

func MergeConfigFile(log logr.Logger, globalOptions interface{}) error {
	log.V(1).Info("using viper config", "config", viper.AllSettings())

	err := viper.Unmarshal(globalOptions)
	if err != nil {
		return err
	}
	log.V(1).Info("unmarshalled options", "config", globalOptions)

	return nil
}


