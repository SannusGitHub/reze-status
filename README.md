# reze-status

## What is this?
"reze-status" is a small server I made during my spare free time which lets me fetch basic information about 
other servers running on my homelab machine (aptly titled 'reze'. sue me, I like Chainsaw Man and Reze).

## ...Why?
Because I want to. I like making things, and I wanted a personal server tracking system with my own spice as
well as comfort added to it.

This program includes the following features:
* Customizable reze-status server parameters:
    * server-name
    * server-image
    * port
    * corsheader
    * refresh-interval
    * return-only-running
* Assigning "servers" to detect running on the machine using a variety of different parameters:
    * name (relayed to the front-end)
    * method ("systemd", "process" & "port" check)
    * unit (used with "systemd" method )
    * match (used with "process" method, supports full path & executable name)
    * port (used with "port" method)
* Returning following information about running server instances to the front-end:
    * server_name
    * server_image
    * running_count
    * total_count
    * servers
        * name
        * running
* A sample settings.yaml file.

This program requires Go 1.2x+ and a Linux host (uses systemd/procfs for detection).

## Installation
Clone repository with:
```
git clone https://github.com/SannusGitHub/reze-status
cd reze-status
```

## Usage
Run the project by doing either of the commands:
```
go run main.go
```

...or, after building:
```
./reze-status
```

## Example API response
```
{
  "server_name": "reze",
  "running_count": 2,
  "total_count": 3,
  "servers": [
    { "name": "minecraft", "running": true },
    { "name": "7dtd", "running": false }
  ]
}
```