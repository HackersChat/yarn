# Setting up `yarnd` with `systemd`

1. Copy `yarnd.sevice` and `yarnd.env` from `deployment/systemd/yarnd.service` to someplace convenient for editing. You will need to decide on values for the following variables and then edit `yarnd.service` and `yarnd.env` accordingly

| Variable      | Meaning                                               |
| ------------- | ----------------------------------------------------- |
| YOUR_POD_URL  | The external URL at which your pod will be accessible |
| YOUR_POD_NAME | The friendly name of your pod                         |
| YARND_USERID  | A system userid                                       |
| YARND_USERGROUP | A system usergroup |
| YOUR_ADMIN_EMAIL_ADDRESS | The email address of your pod's admisnitrator |
| YOUR_ADMIN_NAME | A human name |
| YARND_ADMIN_USERID | A system userid; can be the same as YARND_USERID |
| YARND_EXECUTABLE_LOCATION | For instance, `/usr/local/bin` |
| YARND_ENVIRONMENT_LOCATION | For instance, `/etc/default` |
| YARND_DATA_LOCATION | For instance `/var/local/data` |

2. Put your newly-edited `systemd` unit into your system's `systemd` units directory. From the `yarn` directory cloned from gitea:

```bash
# sudo cp yarnd.service /lib/systemd/system/
```

3. Put yoru newly-edited `yarnd.env` file into YARND_ENVIRONMENT_LOCATION, the location you chose earlier:
```bash
# sudo cp yarnd.env YARND_ENVIRONMENT_LOCATION
```

4. Tell the `systemd` daemon to reload its configuration
```bash
# sudo systemctl daemon-reload
```

5. Tell `systemd` to start the yarn daemon
```bash
# sudo systemctl start yarnd
```

6. Verify the daemon started correctly
```bash
# sudo systemctl status yarnd
```

7. If you want `yarnd` to be restarted when your system reboots, enable the unit
```bash
# sudo systemctl enable yarnd
```
