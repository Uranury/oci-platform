terraform {
  required_version = ">= 1.6.0"

	required_providers {
		oci = {
			source = "oracle/oci"
      version = "~> 7.0"
		}
	}
}

provider "oci" {
	region = "eu-frankfurt-1"
	auth = "SecurityToken"
	config_file_profile = "~/.oci/config"	
}
