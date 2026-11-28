terraform {
	required_providers {
		oci = {
			source = "oracle/oci"
		}
	}
}

provider "oci" {
	region = "eu-frankfurt-1"
	auth = "SecurityToken"
	config_file_profile = "~/.oci/config"	
}
