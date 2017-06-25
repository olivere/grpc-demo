package com.altf4.grpc.client;

import java.util.TimeZone;
import java.util.concurrent.TimeUnit;

/**
 * Created by oliver on 25.06.17.
 */
public class TickerCommand {
    private final ExampleClient client;

    public TickerCommand(ExampleClient client) {
        this.client = client;
    }

    public void run() {
        long intervalNanos = TimeUnit.NANOSECONDS.convert(1, TimeUnit.SECONDS);
        client.ticker(TimeZone.getDefault(), intervalNanos, (String message) -> {
            System.out.println(message);
        });
    }
}
