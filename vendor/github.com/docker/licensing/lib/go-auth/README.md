This is a library for handling JWT authentication tokens

### Generating new test certificates

In the event that the test certificates expire, regenerate them with OpenSSL

    cd jwt/testdata

Generate new root CA cert and private key

    openssl req -newkey rsa:4096 -nodes -keyout root_key.pem -x509 -days 3650 -out root-certs
    

Generate new intermediate CA cert and private key

    openssl req -new -key private-key -out inter.csr
    
    openssl x509 -req -days 3650 -in inter.csr -CA root-certs -CAkey root_key.pem -CAcreateserial -out trusted-cert
    
  
   
