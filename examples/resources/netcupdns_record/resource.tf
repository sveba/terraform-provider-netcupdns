resource "netcupdns_record" "root" {
  destination = "1.2.3.4"
  domainname  = "example.com"
  hostname    = "@"
  type        = "A"
}

resource "netcupdns_record" "mail" {
  destination = "mail.example.com"
  domainname  = "example.com"
  hostname    = "@"
  type        = "MX"
  priority    = "5"
}