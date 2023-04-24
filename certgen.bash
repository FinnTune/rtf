#!/bin/bash

#localhost.crt, localhost.key, and localhost.csr files were created using the following CLI commands:
openssl req  -new  -newkey rsa:2048  -nodes  -keyout localhost.key  -out localhost.csr
openssl  x509  -req  -days 365  -in localhost.csr  -signkey localhost.key  -out localhost.crt

#Syntax for block comment
: <<'END'
Comment description for above commands:
These OpenSSL commands are used to generate a self-signed SSL/TLS certificate for a local development or testing environment. 
The first command generates a private key and a certificate signing request (CSR), 
while the second command creates a self-signed SSL/TLS certificate using the private key and the CSR generated by the first command.

Here's a breakdown of each command and its options:

    openssl req -new -newkey rsa:2048 -nodes -keyout localhost.key -out localhost.csr
        req: generates a certificate signing request
        -new: generates a new certificate request
        -newkey rsa:2048: creates a new RSA key with a length of 2048 bits
        -nodes: specifies that the private key should not be encrypted with a passphrase
        -keyout localhost.key: specifies the filename for the generated private key
        -out localhost.csr: specifies the filename for the generated certificate signing request

    openssl x509 -req -days 365 -in localhost.csr -signkey localhost.key -out localhost.crt
        x509: generates an X.509 certificate
        -req: specifies that the input file is a certificate signing request
        -days 365: sets the validity period of the certificate to 365 days
        -in localhost.csr: specifies the filename of the certificate signing request to be signed
        -signkey localhost.key: specifies the filename of the private key to sign the certificate
        -out localhost.crt: specifies the filename for the generated self-signed certificate

After running these commands, you should have three files in your directory: localhost.key, localhost.csr, and localhost.crt. 
The localhost.key file is your private key, and you should keep it secure. 
The localhost.csr file is your certificate signing request, and you can discard it once you have generated the certificate. 
The localhost.crt file is your self-signed SSL/TLS certificate, which you can use for local development and testing purposes.
END