package dns

import (
	"github.com/csvwolf/ker.go/api"
	"github.com/csvwolf/ker.go/constant"
	"github.com/csvwolf/ker.go/model"
	"github.com/jeessy2/ddns-go/v6/config"
	"github.com/jeessy2/ddns-go/v6/util"
	"strconv"
)

var _ DNS = &Hostker{}

type Hostker struct {
	api     *api.Ker
	DNS     config.DNS
	Domains config.Domains
	TTL     uint32
}

func (h *Hostker) Init(dnsConf *config.DnsConfig, ipv4cache *util.IpCache, ipv6cache *util.IpCache) {
	h.Domains.Ipv4Cache = ipv4cache
	h.Domains.Ipv6Cache = ipv6cache
	h.DNS = dnsConf.DNS
	h.Domains.GetNewIp(dnsConf)
	if dnsConf.TTL == "" {
		// 默认600
		h.TTL = 600
	} else {
		ttl, err := strconv.ParseUint(dnsConf.TTL, 10, 32)
		if err != nil || ttl < 600 {
			util.Log("TTL参数错误，设置为默认值600s！%s", err)
			h.TTL = 600
		} else {
			h.TTL = uint32(ttl)
		}
	}
	h.api = api.New(h.DNS.ID, h.DNS.Secret)
}

func (h *Hostker) AddUpdateDomainRecords() config.Domains {
	h.updateDomainRecordsByRecordType("A")
	h.updateDomainRecordsByRecordType("AAAA")
	return h.Domains
}

func (h *Hostker) updateDomainRecordsByRecordType(recordType string) {
	ipaddr, domains := h.Domains.GetNewIpResult(recordType)
	if ipaddr == "" {
		return
	}

	for _, domain := range domains {
		resp, err := h.api.GetDomainRecords(domain.DomainName)
		if err != nil {
			util.Log("查询域名信息发生异常！%s", err)
			domain.UpdateStatus = config.UpdatedFailed
			continue
		}

		found := false
		for _, r := range resp.Records {
			if r.Type == toAPIRecordType(recordType) && r.Header == domain.GetSubDomain() {
				n := *r
				n.Value = ipaddr
				_, err := h.api.UpdateDomainRecord(domain.DomainName, r, &n)
				if err != nil {
					util.Log("更新域名发生异常！%s", err)
					continue
				}
				found = true
				break
			}
		}

		if !found {
			record := &model.Record{
				Header: domain.GetSubDomain(),
				Type:   toAPIRecordType(recordType),
				Ttl:    h.TTL,
				Value:  ipaddr,
			}
			_, err := h.api.CreateDomainRecord(domain.DomainName, record)
			if err != nil {
				util.Log("更新域名发生异常！%s", err)
				continue
			}
		}
	}
}

func toAPIRecordType(recordType string) constant.DomainType {
	switch recordType {
	case "AAAA":
		return constant.DomainTypeAAAA
	case "A":
		return constant.DomainTypeA
	}
	return ""
}
