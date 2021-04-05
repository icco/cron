package stats

import "testing"

func TestUnmarshalAirplanes(t *testing.T) {
	tests := []string{
		`{ "now" : 1617502620.6, "messages" : 43490633, "aircraft" : [ {"hex":"ad273b","alt_baro":34575,"lat":40.145207,"lon":-74.572012,"nic":8,"rc":186,"seen_pos":38.3,"version":0,"nac_p":8,"sil":2,"sil_type":"unknown","mlat":[],"tisb":[],"messages":12,"seen":26.8,"rssi":-10.2}, {"hex":"a8ebe7","version":0,"sil_type":"unknown","mlat":[],"tisb":[],"messages":9,"seen":133.7,"rssi":-12.4}, {"hex":"06a0e8","alt_baro":27400,"alt_geom":27050,"gs":461.1,"track":45.9,"baro_rate":768,"category":"A5","nav_qnh":1013.6,"nav_altitude_mcp":36992,"nav_heading":54.1,"version":0,"nic_baro":1,"nac_p":9,"nac_v":2,"sil":3,"sil_type":"unknown","mlat":[],"tisb":[],"messages":86,"seen":0.1,"rssi":-8.8}, {"hex":"a0fe19","alt_baro":43000,"alt_geom":42500,"gs":444.3,"track":326.4,"baro_rate":-64,"squawk":"1423","emergency":"none","category":"A2","lat":41.876391,"lon":-73.016031,"nic":8,"rc":186,"seen_pos":15.4,"version":2,"nic_baro":1,"nac_p":10,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":472,"seen":0.2,"rssi":-6.6}, {"hex":"a48315","flight":"DAL2267 ","alt_baro":41000,"alt_geom":40650,"gs":477.4,"track":92.4,"baro_rate":0,"category":"A3","nav_qnh":1013.6,"nav_altitude_mcp":6016,"nav_heading":102.0,"lat":42.716752,"lon":-73.587895,"nic":8,"rc":186,"seen_pos":11.5,"version":2,"nic_baro":1,"nac_p":10,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":151,"seen":5.4,"rssi":-7.3}, {"hex":"a2af3e","flight":"EDV5206 ","alt_baro":21100,"alt_geom":20950,"gs":429.5,"track":98.3,"geom_rate":-1408,"category":"A3","nav_qnh":1012.8,"nav_altitude_mcp":19008,"nav_heading":104.8,"version":2,"nic_baro":1,"nac_p":10,"nac_v":1,"sil":3,"sil_type":"perhour","mlat":[],"tisb":[],"messages":73,"seen":2.1,"rssi":-9.4}, {"hex":"a64c88","alt_baro":29000,"alt_geom":28575,"gs":433.8,"track":12.8,"geom_rate":0,"squawk":"7155","emergency":"none","category":"A2","nav_qnh":1012.8,"nav_altitude_mcp":28992,"lat":41.707826,"lon":-74.127183,"nic":8,"rc":186,"seen_pos":4.4,"version":2,"nic_baro":1,"nac_p":10,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":1067,"seen":1.6,"rssi":-2.0}, {"hex":"4caa41","flight":"EIN9078 ","alt_baro":39000,"alt_geom":38650,"gs":479.4,"track":48.0,"baro_rate":0,"squawk":"3107","emergency":"none","category":"A5","nav_qnh":1012.8,"nav_altitude_mcp":39008,"lat":41.040298,"lon":-73.213135,"nic":8,"rc":186,"seen_pos":39.3,"version":2,"nic_baro":1,"nac_p":9,"nac_v":1,"sil":3,"sil_type":"perhour","mlat":[],"tisb":[],"messages":858,"seen":5.3,"rssi":-5.8}, {"hex":"a6b4f7","version":2,"sil_type":"perhour","mlat":[],"tisb":[],"messages":476,"seen":144.6,"rssi":-10.2}, {"hex":"a4dc9e","version":2,"sil_type":"perhour","mlat":[],"tisb":[],"messages":218,"seen":206.4,"rssi":-11.0}, {"hex":"a8ae2e","category":"A2","version":2,"sil_type":"perhour","mlat":[],"tisb":[],"messages":3123,"seen":107.9,"rssi":-9.8} ] }`,
		`{ "now" : 1617560400.2, "messages" : 45532105, "aircraft" : [ {"hex":"a60efc","nav_qnh":1013.6,"nav_altitude_mcp":45024,"nav_modes":["autopilot","althold","tcas"],"version":0,"nic_baro":1,"nac_p":10,"sil":3,"sil_type":"unknown","mlat":[],"tisb":[],"messages":8,"seen":4.8,"rssi":-14.2}, {"hex":"a26d9c","type":"adsr_icao","flight":"N25582  ","alt_baro":3300,"alt_geom":3175,"gs":72.3,"track":275.6,"geom_rate":448,"category":"A1","lat":40.979097,"lon":-73.246973,"nic":8,"rc":186,"seen_pos":3.0,"nic_baro":0,"nac_p":9,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":87,"seen":3.0,"rssi":-2.6}, {"hex":"ad1f08","alt_baro":"ground","squawk":"1200","emergency":"none","version":0,"mlat":[],"tisb":[],"messages":14,"seen":3.7,"rssi":-11.3}, {"hex":"~274c28","type":"tisb_other","alt_baro":300,"alt_geom":2675,"gs":35.9,"track":329.9,"geom_rate":0,"lat":41.057584,"lon":-73.703842,"nic":6,"rc":926,"seen_pos":7.6,"nac_p":6,"nac_v":7,"sil":2,"sil_type":"unknown","mlat":[],"tisb":["altitude","alt_geom","gs","track","geom_rate","lat","lon","nic","rc","nac_p","nac_v","sil","sil_type"],"messages":22,"seen":7.6,"rssi":-2.6}, {"hex":"3c6570","flight":"DLH418  ","alt_baro":40000,"alt_geom":39600,"gs":463.5,"track":231.7,"baro_rate":-64,"squawk":"1401","emergency":"none","category":"A5","nav_qnh":1013.6,"nav_altitude_mcp":40000,"nav_heading":0.0,"lat":42.014465,"lon":-72.968445,"nic":8,"rc":186,"seen_pos":7.0,"version":2,"nic_baro":1,"nac_p":9,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":311,"seen":0.6,"rssi":-9.3}, {"hex":"71c359","flight":"AAR221  ","alt_baro":17150,"alt_geom":16875,"gs":349.3,"track":315.0,"baro_rate":1856,"squawk":"2777","category":"A5","nav_qnh":1018.4,"nav_altitude_mcp":24992,"nav_heading":329.1,"lat":41.031189,"lon":-73.858826,"nic":8,"rc":186,"seen_pos":3.6,"version":2,"nic_baro":1,"nac_p":9,"nac_v":2,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":132,"seen":2.6,"rssi":-8.2}, {"hex":"a85caa","alt_baro":12900,"alt_geom":12750,"gs":355.3,"track":237.3,"baro_rate":3072,"nav_qnh":1016.8,"nav_altitude_mcp":16992,"lat":40.784317,"lon":-74.231873,"nic":8,"rc":186,"seen_pos":3.7,"version":2,"nic_baro":1,"nac_p":10,"nac_v":4,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":65,"seen":1.4,"rssi":-11.1}, {"hex":"a05ad1","alt_baro":14000,"version":2,"sil_type":"perhour","mlat":[],"tisb":[],"messages":16,"seen":49.8,"rssi":-12.5}, {"hex":"ac84e8","alt_baro":3900,"gs":85.4,"track":205.7,"baro_rate":512,"lat":41.647494,"lon":-73.770702,"nic":0,"rc":0,"seen_pos":5.9,"nac_p":0,"nac_v":0,"sil":0,"sil_type":"unknown","mlat":["gs","track","baro_rate","lat","lon","nic","rc","nac_p","nac_v","sil","sil_type"],"tisb":[],"messages":124,"seen":1.1,"rssi":-6.6}, {"hex":"a6cb48","flight":"JBU222  ","alt_baro":35025,"alt_geom":34575,"gs":435.9,"track":35.5,"baro_rate":64,"squawk":"3736","emergency":"none","category":"A3","nav_qnh":1013.6,"nav_altitude_mcp":27008,"nav_heading":0.0,"lat":40.372925,"lon":-73.898438,"nic":8,"rc":186,"seen_pos":3.1,"version":2,"nic_baro":1,"nac_p":9,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":412,"seen":0.5,"rssi":-11.3}, {"hex":"406590","flight":"BAW3581 ","alt_baro":36000,"alt_geom":35475,"gs":479.2,"track":233.7,"baro_rate":0,"squawk":"0716","emergency":"none","category":"A5","nav_qnh":1012.8,"nav_altitude_mcp":36000,"nav_heading":253.1,"lat":41.475577,"lon":-73.236460,"nic":8,"rc":186,"seen_pos":0.5,"version":2,"nic_baro":1,"nac_p":9,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":681,"seen":0.0,"rssi":-4.5}, {"hex":"a99386","type":"adsr_icao","flight":"N716NC  ","alt_baro":700,"alt_geom":625,"gs":62.6,"track":289.6,"geom_rate":-192,"category":"A1","lat":41.669699,"lon":-74.128844,"nic":8,"rc":186,"seen_pos":0.3,"nic_baro":0,"nac_p":9,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":182,"seen":0.3,"rssi":-4.2}, {"hex":"a327cc","flight":"SIS302  ","alt_baro":35250,"alt_geom":34750,"gs":446.9,"track":249.3,"geom_rate":1856,"squawk":"1727","emergency":"none","category":"A2","nav_qnh":1012.8,"nav_altitude_mcp":40000,"nav_heading":270.0,"lat":40.252304,"lon":-74.323547,"nic":8,"rc":186,"seen_pos":2.0,"version":2,"nic_baro":1,"nac_p":10,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":549,"seen":0.1,"rssi":-10.3}, {"hex":"a2dda1","flight":"JBU465  ","alt_baro":35450,"alt_geom":34925,"gs":436.1,"track":235.2,"baro_rate":1024,"squawk":"3560","emergency":"none","category":"A3","nav_qnh":1013.6,"nav_altitude_mcp":38016,"nav_modes":["autopilot","tcas"],"lat":41.488425,"lon":-73.489976,"nic":8,"rc":186,"seen_pos":0.9,"version":2,"nic_baro":1,"nac_p":10,"nac_v":2,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":1932,"seen":0.1,"rssi":-4.2}, {"hex":"abf17f","flight":"AAL1163 ","alt_baro":37000,"alt_geom":36575,"gs":447.0,"track":39.3,"baro_rate":0,"squawk":"0514","emergency":"none","category":"A3","nav_qnh":1013.6,"nav_altitude_mcp":36992,"nav_heading":57.7,"lat":40.379974,"lon":-74.305054,"nic":8,"rc":186,"seen_pos":4.4,"version":2,"nic_baro":1,"nac_p":9,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":468,"seen":1.2,"rssi":-12.7}, {"hex":"a39269","alt_baro":9425,"squawk":"3005","lat":41.725388,"lon":-73.421131,"nic":8,"rc":186,"seen_pos":21.3,"version":0,"nac_p":8,"sil":2,"sil_type":"unknown","mlat":[],"tisb":[],"messages":37,"seen":21.3,"rssi":-14.4}, {"hex":"a11fb9","type":"adsr_icao","flight":"N1713V  ","alt_baro":4300,"alt_geom":4200,"gs":84.2,"track":283.0,"geom_rate":-704,"category":"A1","lat":41.646656,"lon":-74.162314,"nic":8,"rc":186,"seen_pos":0.3,"nic_baro":0,"nac_p":9,"nac_v":2,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":292,"seen":0.3,"rssi":-4.2}, {"hex":"a3f0bc","category":"A3","version":2,"sil_type":"perhour","mlat":[],"tisb":[],"messages":49,"seen":142.2,"rssi":-12.0}, {"hex":"a18fed","version":0,"sil_type":"unknown","mlat":[],"tisb":[],"messages":6,"seen":168.9,"rssi":-14.9}, {"hex":"71c041","flight":"KAL082  ","alt_baro":29800,"alt_geom":29250,"gs":425.4,"track":4.7,"baro_rate":960,"squawk":"3303","emergency":"none","category":"A5","nav_qnh":1012.8,"nav_altitude_mcp":31008,"nav_heading":14.8,"lat":41.737014,"lon":-73.367466,"nic":8,"rc":186,"seen_pos":0.3,"version":2,"nic_baro":1,"nac_p":9,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":2050,"seen":0.3,"rssi":-6.8}, {"hex":"a71b43","alt_baro":33600,"alt_geom":33125,"gs":434.6,"track":36.6,"baro_rate":-2240,"category":"A3","lat":40.558640,"lon":-73.738281,"nic":8,"rc":186,"seen_pos":22.7,"version":2,"nac_v":1,"sil_type":"perhour","mlat":[],"tisb":[],"messages":191,"seen":22.2,"rssi":-11.7}, {"hex":"a2b141","flight":"GPD27   ","alt_baro":15300,"alt_geom":15075,"gs":190.1,"track":354.0,"baro_rate":1088,"squawk":"1762","emergency":"none","category":"A1","lat":41.567378,"lon":-73.781816,"nic":9,"rc":75,"seen_pos":0.0,"version":2,"nic_baro":1,"nac_p":10,"nac_v":2,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":1499,"seen":0.0,"rssi":-2.2}, {"hex":"a71957","flight":"N5569Q  ","alt_baro":4175,"alt_geom":4075,"gs":110.4,"track":35.4,"baro_rate":960,"squawk":"1200","emergency":"none","category":"A1","nav_qnh":1016.8,"nav_altitude_mcp":1504,"nav_modes":[],"lat":41.613556,"lon":-74.022945,"nic":9,"rc":75,"seen_pos":0.1,"version":2,"nic_baro":1,"nac_p":10,"nac_v":2,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":794,"seen":0.1,"rssi":-2.4}, {"hex":"~c3b477","type":"tisb_other","sil_type":"unknown","mlat":["nac_v"],"tisb":[],"messages":15,"seen":254.7,"rssi":-49.5}, {"hex":"a263f8","flight":"N253PR  ","alt_baro":3450,"alt_geom":3425,"gs":147.2,"track":182.7,"baro_rate":192,"category":"A1","nav_qnh":1017.6,"lat":41.338108,"lon":-74.358840,"nic":9,"rc":75,"seen_pos":3.7,"version":2,"nic_baro":1,"nac_p":10,"nac_v":2,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":73,"seen":3.3,"rssi":-10.6}, {"hex":"a74909","alt_baro":24425,"alt_geom":22950,"gs":388.6,"track":242.9,"geom_rate":2432,"category":"A2","version":2,"nac_v":1,"sil_type":"perhour","mlat":[],"tisb":[],"messages":251,"seen":17.0,"rssi":-14.4}, {"hex":"a00189","flight":"N1RR    ","alt_baro":3425,"alt_geom":3275,"gs":134.8,"track":213.8,"baro_rate":256,"squawk":"1200","emergency":"none","category":"A1","nav_qnh":1018.4,"nav_altitude_mcp":3488,"lat":41.507977,"lon":-73.871687,"nic":9,"rc":75,"seen_pos":0.0,"version":2,"nic_baro":1,"nac_p":10,"nac_v":2,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":1253,"seen":0.0,"rssi":-3.1}, {"hex":"a7e152","flight":"DAL2561 ","alt_baro":32000,"alt_geom":31500,"gs":427.1,"track":228.3,"baro_rate":0,"squawk":"3443","emergency":"none","category":"A3","nav_qnh":1013.6,"nav_altitude_mcp":32000,"nav_heading":251.0,"lat":41.322556,"lon":-73.967163,"nic":8,"rc":186,"seen_pos":0.3,"version":2,"nic_baro":1,"nac_p":9,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":3413,"seen":0.3,"rssi":-3.0}, {"hex":"ac9936","category":"A1","version":2,"sil_type":"perhour","mlat":[],"tisb":[],"messages":67,"seen":58.6,"rssi":-10.7}, {"hex":"a575a2","version":2,"sil_type":"perhour","mlat":[],"tisb":[],"messages":33,"seen":113.8,"rssi":-11.2}, {"hex":"a23bfb","category":"A3","version":2,"sil_type":"perhour","mlat":[],"tisb":[],"messages":119,"seen":287.9,"rssi":-12.3}, {"hex":"a3db5e","flight":"DAL2124 ","alt_baro":26625,"alt_geom":26200,"gs":426.0,"track":51.2,"baro_rate":-1152,"squawk":"3136","emergency":"none","category":"A3","nav_altitude_mcp":23008,"nav_heading":0.0,"lat":40.834366,"lon":-73.480058,"nic":8,"rc":186,"seen_pos":4.3,"version":2,"nic_baro":1,"nac_p":9,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":2245,"seen":2.1,"rssi":-11.1}, {"hex":"a0a545","flight":"UAL988  ","alt_baro":31775,"alt_geom":31325,"gs":423.1,"track":238.8,"geom_rate":-1216,"squawk":"0734","emergency":"none","category":"A5","nav_qnh":1012.8,"nav_altitude_mcp":28000,"nav_heading":246.8,"nav_modes":["autopilot","vnav","lnav","tcas"],"lat":41.076903,"lon":-74.490800,"nic":8,"rc":186,"seen_pos":15.4,"version":2,"nic_baro":1,"nac_p":9,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":5867,"seen":7.2,"rssi":-10.2}, {"hex":"a7a59f","category":"A3","version":2,"sil_type":"perhour","mlat":[],"tisb":[],"messages":1823,"seen":69.9,"rssi":-9.7}, {"hex":"a39e7e","category":"A3","version":2,"sil_type":"perhour","mlat":[],"tisb":[],"messages":2486,"seen":144.5,"rssi":-9.8}, {"hex":"acb9bf","alt_baro":28000,"alt_geom":27650,"gs":444.0,"track":238.4,"geom_rate":0,"squawk":"2027","emergency":"none","category":"A2","nav_qnh":1012.8,"nav_altitude_mcp":28000,"lat":40.915924,"lon":-74.837158,"nic":8,"rc":186,"seen_pos":3.6,"version":2,"nic_baro":1,"nac_p":10,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":4048,"seen":1.2,"rssi":-13.3}, {"hex":"ad7e93","alt_baro":16400,"alt_geom":16175,"gs":253.6,"track":228.5,"baro_rate":64,"squawk":"1110","emergency":"none","category":"A1","lat":40.644470,"lon":-74.611450,"nic":9,"rc":75,"seen_pos":1.4,"version":2,"nic_baro":1,"nac_p":10,"nac_v":2,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":3026,"seen":0.8,"rssi":-9.5}, {"hex":"3965a1","alt_baro":29475,"alt_geom":29900,"gs":478.2,"track":238.3,"baro_rate":-1664,"category":"A5","lat":40.858573,"lon":-74.958843,"nic":8,"rc":186,"seen_pos":56.7,"version":2,"nac_v":1,"sil_type":"perhour","mlat":[],"tisb":[],"messages":3721,"seen":30.0,"rssi":-13.4}, {"hex":"47c1c2","flight":"SAS909  ","alt_baro":5875,"alt_geom":5825,"gs":280.0,"track":189.7,"baro_rate":-704,"squawk":"0731","emergency":"none","category":"A5","nav_qnh":1016.8,"nav_altitude_mcp":4992,"nav_heading":205.3,"lat":40.915041,"lon":-74.511899,"nic":8,"rc":186,"seen_pos":36.6,"version":2,"nic_baro":1,"nac_p":9,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":1135,"seen":23.4,"rssi":-12.4}, {"hex":"a56f84","category":"A1","version":2,"sil_type":"perhour","mlat":[],"tisb":[],"messages":215,"seen":250.2,"rssi":-11.2}, {"hex":"a7fb1c","flight":"JIA5616 ","alt_baro":36000,"alt_geom":35675,"gs":458.8,"track":231.3,"geom_rate":64,"squawk":"3442","emergency":"none","category":"A3","nav_qnh":1012.8,"nav_altitude_mcp":36000,"nav_heading":249.6,"lat":40.599152,"lon":-74.715759,"nic":8,"rc":186,"seen_pos":4.5,"version":2,"nic_baro":1,"nac_p":10,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":7207,"seen":3.6,"rssi":-12.0}, {"hex":"aa737d","gs":286.5,"track":352.4,"baro_rate":-448,"category":"A3","version":2,"nac_v":1,"sil_type":"perhour","mlat":[],"tisb":[],"messages":2570,"seen":59.9,"rssi":-12.4}, {"hex":"a023c3","flight":"N108PT  ","alt_baro":2800,"alt_geom":2750,"gs":99.1,"track":98.7,"geom_rate":0,"category":"A1","lat":41.039243,"lon":-74.512773,"nic":8,"rc":186,"seen_pos":12.3,"nic_baro":0,"nac_p":9,"nac_v":1,"sil":3,"sil_type":"perhour","gva":2,"sda":2,"mlat":[],"tisb":[],"messages":634,"seen":12.3,"rssi":-4.1} ] }`,
	}

	for _, v := range tests {
		if _, err := unmarshalAirplanes([]byte(v)); err != nil {
			t.Errorf("expected no error, got %+v", err)
		}
	}
}