terraform {
  required_providers {
    netcupdns = {
      source = "sveba/netcupdns"
    }
  }
}

provider "netcupdns" {
  customer_number = "123456"
  key             = "theApiKey"
  password        = "theApiPass"
}