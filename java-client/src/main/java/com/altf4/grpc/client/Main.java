package com.altf4.grpc.client;

import io.grpc.ManagedChannelBuilder;
import org.apache.commons.cli.*;

import java.util.TimeZone;
import java.util.concurrent.TimeUnit;

/**
 * Created by oliver on 24.06.17.
 */
public class Main {
    public static void main(String []args) throws InterruptedException {
        // Prepare and parse command line
        CommandLineParser parser = new DefaultParser();
        Options options = new Options();
        options.addOption("help", "h", false,"Print help");
        options.addOption("addr", true, "Server address, e.g. localhost:10000");
        options.addOption("tls", false, "Enable TLS");
        options.addOption("serverName", true, "Server to check the certificate");
        options.addOption("caFile", true, "Certificate file in e.g. in PEM format");
        options.addOption("interval", true, "Time interval between ticker responses");
        HelpFormatter helpFormatter = new HelpFormatter();
        CommandLine cmd = null;
        try {
            cmd = parser.parse(options, args);
        } catch (ParseException e) {
            helpFormatter.printHelp("java-client [options] <hello | ticker>", options);
            System.exit(1);
        }

        if (cmd.hasOption("help") || cmd.hasOption('h')) {
            helpFormatter.printHelp("java-client [options] <hello | ticker>", options);
            System.exit(0);
        }

        String serverAddress = "localhost:10000";
        if (cmd.hasOption("addr")) {
            serverAddress = cmd.getOptionValue("addr");
        }

        // Parse server address
        String host;
        int port = 0;
        int colonPos = serverAddress.indexOf(':');
        if (colonPos >= 0 && serverAddress.indexOf(':', colonPos+1) == -1) {
            host = serverAddress.substring(0, colonPos);
            String portString = serverAddress.substring(colonPos+1);
            if (portString != null && portString != "") {
                try {
                    port = Integer.parseInt(portString);
                } catch (NumberFormatException e) {
                    throw new IllegalArgumentException("Unparseable port number in: " + serverAddress);
                }
            }
        } else {
            host = serverAddress;
        }

        // Create a managed channel to the server
        ManagedChannelBuilder channelBuilder = ManagedChannelBuilder.forAddress(host, port)
                .usePlaintext(true);
        ExampleClient client = new ExampleClient(channelBuilder);

        switch (cmd.getArgList().stream().findFirst().orElse("hello")) {
            case "hello":
                new HelloCommand(client).run();
                break;
            case "ticker":
                new TickerCommand(client).run();
                break;
        }

        client.shutdown();
    }
}
