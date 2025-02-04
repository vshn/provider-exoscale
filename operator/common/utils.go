package common

import (
	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
)

var (
	ZoneTranslation = map[exoscalev1.Zone]exoscalesdk.Endpoint{
		"CH-DK-2":  exoscalesdk.CHDk2,
		"CH-GVA-2": exoscalesdk.CHGva2,
		"DE-FRA-1": exoscalesdk.DEFra1,
		"DE-MUC-1": exoscalesdk.DEMuc1,
		"AT-VIE-1": exoscalesdk.ATVie1,
		"AT-VIE-2": exoscalesdk.ATVie2,
		"BG-SOF-1": exoscalesdk.BGSof1,
		"ch-dk-2":  exoscalesdk.CHDk2,
		"ch-gva-2": exoscalesdk.CHGva2,
		"de-fra-1": exoscalesdk.DEFra1,
		"de-muc-1": exoscalesdk.DEMuc1,
		"at-vie-1": exoscalesdk.ATVie1,
		"at-vie-2": exoscalesdk.ATVie2,
		"bg-sof-1": exoscalesdk.BGSof1,
	}
)
