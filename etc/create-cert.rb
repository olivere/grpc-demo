#!/usr/bin/env ruby

# Usage ./create-cert.rb grpc-demo.go
# => Outputs: grpc-demo.go.pem and grpc-demo.go.key

name = ARGV[0]

domain = "*.#{name}" # "*.grpc-demo.go"
subjectAltDomains = [ domain, name ] # [ domain, "grpc-demo.go" ]

require 'openssl'
puts "Generating public and private keys..."
key = OpenSSL::PKey::RSA.new(2048)

subject = "/C=DE/L=Munich/O=GrpcDemo/CN=#{domain}"

cert = OpenSSL::X509::Certificate.new
cert.subject = cert.issuer = OpenSSL::X509::Name.parse(subject)

cert.not_before = Time.now
cert.not_after = Time.now + 10*365*24*60*60
cert.public_key = key.public_key
cert.serial = 0x0
cert.version = 2

puts "Signing certificate..."
ef = OpenSSL::X509::ExtensionFactory.new
ef.subject_certificate = ef.issuer_certificate = cert

cert.extensions = [
  # Set to CA:FALSE if you don't want this cert to "be" a CA
  ef.create_extension("basicConstraints","CA:TRUE", true),
  ef.create_extension("subjectKeyIdentifier", "hash")
]
cert.add_extension ef.create_extension("authorityKeyIdentifier",
                                 "keyid:always,issuer:always")
cert.add_extension ef.create_extension("subjectAltName", subjectAltDomains.map { |d| "DNS: #{d}" }.join(','))


cert.sign key, OpenSSL::Digest::SHA1.new

File.open("#{name}.pem", "w") { |f| f.write(cert.to_pem) }
File.open("#{name}.key", "w") { |f| f.write(key.to_s) }

puts "Now run (something like) this on MacOS:\nsudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain #{name}.pem"
