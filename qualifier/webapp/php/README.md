## Run

### on ISUCON AMI

```
sudo cp /etc/nginx/nginx.conf{,.orig}
sudo cp ./nginx.conf /etc/nginx/
sudo /etc/init.d/nginx reload
```

### Local

```
foreman start
open http://localhost:8080/
```
