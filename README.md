# Monero fastsync service
The service provides caching and precalculation for blocks potentially containing particular wallet's transactions. 
It eases CPU load on mobile devices during synchronization and speeds up blockchain rescan operations.

## Structure
The service consists of two components:
* `syncer` - synchronizes the blockchain with DB
* `fsd` - serves synchronization requests from wallets 

## How to run
At first, set up `postgresql` DB and create database with the [following](scripts/create_db.sql) structure.

Download the repo:
```
go get -u -v github.com/exantech/monero-fastsync/
```

Build `syncer`:
```
go build github.com/exantech/monero-fastsync/cmd/syncer
```

Make config file from the [template](configs/syncer.yml), and run it:
```
./syncer -config /path/to/syncer.yml
```

Wait until your DB is synchronized with blockchain.

Build `fsd`:
```
go build github.com/exantech/monero-fastsync/cmd/fsd
```

Make config [file](configs/fsd.yml) and run it:
```
./fsd -config /path/to/fsd.yml
```

`fsd` itself has only one endpoint - `/fastsync.bin` where fastsync clients send requests to. All other requests to monero node are proxied to real node with `nginx`.
Get `nginx` [config](configs/fastsync.conf) template, substitute fastsync and monero nodes urls, place it to `/etc/nginx/sites-available`, make a symlink:
```
sudo ln -s /etc/nginx/sites-available/fastsync.conf /etc/nginx/sites-enabled/fastsync.conf
```
and run:
```
nginx -t
```

if it says the config is okay you may reload web server:
```
sudo service nginx reload
```

Now you may apply fastsync patch (top commit on [patch-v0.13/fastsync](https://github.com/exantech/monero/commits/patch-v0.13/fastsync)) to your monero wallet or get version supporting it and run your wallet. 
To get fastsync working you need to set up refresh type with console command:
```
[wallet 53MGew (out of sync)]: set refresh-type fastsync
```
or in your GUI.
