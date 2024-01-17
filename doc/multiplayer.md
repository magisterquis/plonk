Multiplayer Setup
=================
Plonk's comes with multiplayer support out of the box.  It is slightly more
complicated to set up, but not particularly hard.  The changes needed are

1. Make a group for access control and put operators in it
2. Make Plonk's binary accessible to authorized operators
3. Make Plonk's directory accessible to authorized operators

Access to the server is via a Unix socket; membership in the socket's group
determines access to Plonk.

Quickstart - Multiplayer (OpenBSD)
----------------------------------
```sh
# We'll use a group, plonk_ops, to control access to Plonk:
groupinfo plonk_ops || doas groupadd plonk_ops
doas usermod -G plonk_ops "$(id -un)" # And other users
# Logout/login for the group change to take effect.

# Instead of plonk using $HOME/plonk.d/ for its working files, we'll use
# /opt/plonk.d, which is more multiplayer-friendly.
doas mkdir -p /opt/plonk.d
doas chgrp plonk_ops /opt/plonk.d
doas chmod g+rwx /opt/plonk.d

# Make available the plonk binary, changing its default directory to
# /opt/plonk.d
go install -ldflags '-X main.DefaultDir=/opt/plonk.d' github.com/magisterquis/plonk@v0.0.1-beta.1
doas mv "$(which plonk)" /usr/local/bin

# Optionally, redirect inbound TCP port 443 to 4433, so we don't need to bind
# to a low port.  Without this, Plonk either must be run suid (dangerous) or
# won't be able to use Let's Encrypt.
if ! doas cat /etc/pf.conf | grep -q 'rdr-to 127.0.0.1 port 4433'; then
        echo 'pass in on egress inet proto tcp to (egress) port 443 rdr-to 127.0.0.1 port 4433' | 
        doas tee -a /etc/pf.conf
        doas pfctl -vf /etc/pf.conf -n # If it looks right, re-run without the -n
fi

# Start it going.  Adjust accordingly if not using PF's rdr-to.
nohup plonk -server -https-address 127.0.0.1:4433 >/dev/null 2>&1 &  # Bonus points for -letsencrypt-domain, too

# Did it work?
ls -lart "/opt/plonk.d/"            # Directory exists?
tail "/opt/plonk.d/log.json"        # Log looks ok?
curl -svk https://127.0.0.1:4433/c  # Implant generation works?

# From here, usage is the same as for singleplayer ops.
```

Quickstart - Multiplayer (Linux)
--------------------------------
This is very similar to [OpenBSD](#quickstart---multiplayer-openbsd), except
we trade ease of binding to 443 for more complicated file permissions.
```sh
# We'll use a group, plonk_ops, to control access to Plonk:
grep -q plonk_ops /etc/group || sudo groupadd plonk_ops
sudo usermod -aG plonk_ops "$(id -un)" # And other users
# Logout/login for the group change to take effect.

# Instead of plonk using $HOME/plonk.d/ for its working files, we'll use
# /opt/plonk.d, which is more multiplayer-friendly.
sudo mkdir -p /opt/plonk.d
sudo chgrp plonk_ops /opt/plonk.d
sudo chmod g+rwx /opt/plonk.d

# Make available the plonk binary, changing its default directory to
# /opt/plonk.d and making it run with our new group as well as the capability
# to bind to low ports.
go install -ldflags '-X main.DefaultDir=/opt/plonk.d' github.com/magisterquis/plonk@v0.0.1-beta.1
sudo mv "$(which plonk)" /usr/local/bin
sudo chgrp plonk_ops /usr/local/bin/plonk
sudo setcap cap_net_bind_service+ep /usr/local/bin/plonk
sudo chmod g+s /usr/local/bin/plonk

# Start it going.
nohup plonk -server >/dev/null 2>&1 &  # Bonus points for -letsencrypt-domain, too

# Did it work?
ls -lart "/opt/plonk.d/"           # Directory exists?
tail "/opt/plonk.d/log.json"       # Log looks ok?
curl -svk https://127.0.0.1:443/c  # Implant generation works?

# From here, usage is the same as for singleplayer ops.
```
