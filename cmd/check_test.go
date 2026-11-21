package cmd

import (
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

func TestDesignHosts(t *testing.T) {
	tests := []struct {
		name string
		set  any
		want []string
	}{
		{"unset", nil, nil},
		{"comma-separated env string", "figma.com, zeplin.io", []string{"figma.com", "zeplin.io"}},
		{"yaml list", []string{"figma.com", "zeplin.io"}, []string{"figma.com", "zeplin.io"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev := viper.Get("design_hosts")
			viper.Set("design_hosts", tt.set)
			t.Cleanup(func() { viper.Set("design_hosts", prev) })
			if got := designHosts(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("designHosts() = %v, want %v", got, tt.want)
			}
		})
	}
}
