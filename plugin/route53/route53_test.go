package route53

import (
	"context"
	"testing"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/miekg/dns"
)

type mockedRoute53 struct {
	route53iface.Route53API
}

func (mockedRoute53) ListResourceRecordSets(input *route53.ListResourceRecordSetsInput) (*route53.ListResourceRecordSetsOutput, error) {
	var value string
	switch aws.StringValue(input.StartRecordType) {
	case "A":
		value = "10.2.3.4"
	case "AAAA":
		value = "2001:db8:85a3::8a2e:370:7334"
	case "PTR":
		value = "ptr.example.org"
	}
	return &route53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*route53.ResourceRecordSet{
			{
				ResourceRecords: []*route53.ResourceRecord{
					{
						Value: aws.String(value),
					},
				},
			},
		},
	}, nil
}

func TestRoute53(t *testing.T) {
	r := Route53{
		zones:  []string{"example.org."},
		keys:   map[string]string{"example.org.": "1234567890"},
		client: mockedRoute53{},
	}

	tests := []struct {
		qname         string
		qtype         uint16
		expectedCode  int
		expectedReply []string // ownernames for the records in the additional section.
		expectedErr   error
	}{
		{
			qname:         "example.org",
			qtype:         dns.TypeA,
			expectedCode:  dns.RcodeSuccess,
			expectedReply: []string{"10.2.3.4"},
			expectedErr:   nil,
		},
		{
			qname:         "example.org",
			qtype:         dns.TypeAAAA,
			expectedCode:  dns.RcodeSuccess,
			expectedReply: []string{"2001:db8:85a3::8a2e:370:7334"},
			expectedErr:   nil,
		},
		{
			qname:         "example.org",
			qtype:         dns.TypePTR,
			expectedCode:  dns.RcodeSuccess,
			expectedReply: []string{"ptr.example.org"},
			expectedErr:   nil,
		},
	}

	ctx := context.TODO()

	for i, tc := range tests {
		req := new(dns.Msg)
		req.SetQuestion(dns.Fqdn(tc.qname), tc.qtype)

		rec := dnstest.NewRecorder(&test.ResponseWriter{})
		code, err := r.ServeDNS(ctx, rec, req)

		if err != tc.expectedErr {
			t.Errorf("Test %d: Expected error %v, but got %v", i, tc.expectedErr, err)
		}
		if code != int(tc.expectedCode) {
			t.Errorf("Test %d: Expected status code %d, but got %d", i, tc.expectedCode, code)
		}
		if len(tc.expectedReply) != 0 {
			for i, expected := range tc.expectedReply {
				var actual string
				switch tc.qtype {
				case dns.TypeA:
					actual = rec.Msg.Answer[i].(*dns.A).A.String()
				case dns.TypeAAAA:
					actual = rec.Msg.Answer[i].(*dns.AAAA).AAAA.String()
				case dns.TypePTR:
					actual = rec.Msg.Answer[i].(*dns.PTR).Ptr
				}
				if actual != expected {
					t.Errorf("Test %d: Expected answer %s, but got %s", i, expected, actual)
				}
			}
		}
	}
}
