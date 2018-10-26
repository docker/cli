package engine

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/internal/licenseutils"
	"github.com/docker/licensing/model"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestSubscriptionContextWrite(t *testing.T) {
	cases := []struct {
		context  formatter.Context
		expected string
	}{
		// Errors
		{
			formatter.Context{Format: "{{InvalidFunction}}"},
			`Template parsing error: template: :1: function "InvalidFunction" not defined
`,
		},
		{
			formatter.Context{Format: "{{nil}}"},
			`Template parsing error: template: :1:2: executing "" at <nil>: nil is not a command
`,
		},
		// Table format
		{
			formatter.Context{Format: NewSubscriptionsFormat("table", false)},
			`NUM                 OWNER               PRODUCT ID          EXPIRES                         PRICING COMPONENTS
1                   owner1              productid1          2020-01-01 10:00:00 +0000 UTC   compstring
2                   owner2              productid2          2020-01-01 10:00:00 +0000 UTC   compstring
`,
		},
		{
			formatter.Context{Format: NewSubscriptionsFormat("table", true)},
			`1:License Name: name1	Quantity: 10 nodes	Expiration date: 2020-01-01
2:License Name: name2	Quantity: 20 nodes	Expiration date: 2020-01-01
`,
		},
		{
			formatter.Context{Format: NewSubscriptionsFormat("table {{.Owner}}", false)},
			`OWNER
owner1
owner2
`,
		},
		{
			formatter.Context{Format: NewSubscriptionsFormat("table {{.Owner}}", true)},
			`OWNER
owner1
owner2
`,
		},
		// Raw Format
		{
			formatter.Context{Format: NewSubscriptionsFormat("raw", false)},
			`license: id1
name: name1
owner: owner1
components: compstring

license: id2
name: name2
owner: owner2
components: compstring

`,
		},
		{
			formatter.Context{Format: NewSubscriptionsFormat("raw", true)},
			`license: id1
license: id2
`,
		},
		// Custom Format
		{
			formatter.Context{Format: NewSubscriptionsFormat("{{.Owner}}", false)},
			`owner1
owner2
`,
		},
	}

	expiration, _ := time.Parse(time.RFC822, "01 Jan 20 10:00 UTC")

	for _, testcase := range cases {
		subscriptions := []licenseutils.LicenseDisplay{
			{
				Num:   1,
				Owner: "owner1",
				Subscription: model.Subscription{
					ID:        "id1",
					Name:      "name1",
					ProductID: "productid1",
					Expires:   &expiration,
					PricingComponents: model.PricingComponents{
						&model.SubscriptionPricingComponent{
							Name:  "nodes",
							Value: 10,
						},
					},
				},
				ComponentsString: "compstring",
			},
			{
				Num:   2,
				Owner: "owner2",
				Subscription: model.Subscription{
					ID:        "id2",
					Name:      "name2",
					ProductID: "productid2",
					Expires:   &expiration,
					PricingComponents: model.PricingComponents{
						&model.SubscriptionPricingComponent{
							Name:  "nodes",
							Value: 20,
						},
					},
				},
				ComponentsString: "compstring",
			},
		}
		out := &bytes.Buffer{}
		testcase.context.Output = out
		err := SubscriptionsWrite(testcase.context, subscriptions)
		if err != nil {
			assert.Error(t, err, testcase.expected)
		} else {
			assert.Check(t, is.Equal(testcase.expected, out.String()))
		}
	}
}

func TestSubscriptionContextWriteJSON(t *testing.T) {
	expiration, _ := time.Parse(time.RFC822, "01 Jan 20 10:00 UTC")
	subscriptions := []licenseutils.LicenseDisplay{
		{
			Num:   1,
			Owner: "owner1",
			Subscription: model.Subscription{
				ID:        "id1",
				Name:      "name1",
				ProductID: "productid1",
				Expires:   &expiration,
				PricingComponents: model.PricingComponents{
					&model.SubscriptionPricingComponent{
						Name:  "nodes",
						Value: 10,
					},
				},
			},
			ComponentsString: "compstring",
		},
		{
			Num:   2,
			Owner: "owner2",
			Subscription: model.Subscription{
				ID:        "id2",
				Name:      "name2",
				ProductID: "productid2",
				Expires:   &expiration,
				PricingComponents: model.PricingComponents{
					&model.SubscriptionPricingComponent{
						Name:  "nodes",
						Value: 20,
					},
				},
			},
			ComponentsString: "compstring",
		},
	}
	expectedJSONs := []map[string]interface{}{
		{
			"Owner":            "owner1",
			"ComponentsString": "compstring",
			"Expires":          "2020-01-01T10:00:00Z",
			"DockerID":         "",
			"Eusa":             nil,
			"ID":               "id1",
			"Start":            nil,
			"Name":             "name1",
			"Num":              float64(1),
			"PricingComponents": []interface{}{
				map[string]interface{}{
					"name":  "nodes",
					"value": float64(10),
				},
			},
			"ProductID":         "productid1",
			"ProductRatePlan":   "",
			"ProductRatePlanID": "",
			"State":             "",
			"Summary":           "License Name: name1\tQuantity: 10 nodes\tExpiration date: 2020-01-01",
		},
		{
			"Owner":            "owner2",
			"ComponentsString": "compstring",
			"Expires":          "2020-01-01T10:00:00Z",
			"DockerID":         "",
			"Eusa":             nil,
			"ID":               "id2",
			"Start":            nil,
			"Name":             "name2",
			"Num":              float64(2),
			"PricingComponents": []interface{}{
				map[string]interface{}{
					"name":  "nodes",
					"value": float64(20),
				},
			},
			"ProductID":         "productid2",
			"ProductRatePlan":   "",
			"ProductRatePlanID": "",
			"State":             "",
			"Summary":           "License Name: name2\tQuantity: 20 nodes\tExpiration date: 2020-01-01",
		},
	}

	out := &bytes.Buffer{}
	err := SubscriptionsWrite(formatter.Context{Format: "{{json .}}", Output: out}, subscriptions)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatal(err)
		}
		assert.Check(t, is.DeepEqual(expectedJSONs[i], m))
	}
}

func TestSubscriptionContextWriteJSONField(t *testing.T) {
	subscriptions := []licenseutils.LicenseDisplay{
		{Num: 1, Owner: "owner1"},
		{Num: 2, Owner: "owner2"},
	}
	out := &bytes.Buffer{}
	err := SubscriptionsWrite(formatter.Context{Format: "{{json .Owner}}", Output: out}, subscriptions)
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var s string
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			t.Fatal(err)
		}
		assert.Check(t, is.Equal(subscriptions[i].Owner, s))
	}
}
