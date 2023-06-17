package metrics

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCounter_FromString(t *testing.T) {
	type want struct {
		counter Counter
		wantErr bool
	}
	tests := []struct {
		name  string
		input string
		want  want
	}{
		{
			name:  "Counter from string ok",
			input: "2",
			want: want{
				counter: 2,
			},
		},
		{
			name:  "Counter from string bad",
			input: "2s",
			want: want{
				wantErr: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var counter Counter
			err := counter.FromString(tt.input)

			if !tt.want.wantErr {
				require.NoError(t, err)
				assert.Equal(t, tt.want.counter, counter)
				return
			}

			require.Error(t, err)
		})
	}
}

func TestGauge_FromString(t *testing.T) {
	type want struct {
		gauge   Gauge
		wantErr bool
	}
	tests := []struct {
		name  string
		input string
		want  want
	}{
		{
			name:  "Gauge from string ok",
			input: "-42.24",
			want: want{
				gauge: -42.24,
			},
		},
		{
			name:  "Gauge from string bad",
			input: "-42.2s",
			want: want{
				wantErr: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gauge Gauge
			err := gauge.FromString(tt.input)

			if !tt.want.wantErr {
				require.NoError(t, err)
				assert.Equal(t, tt.want.gauge, gauge)
				return
			}

			require.Error(t, err)
		})
	}
}
