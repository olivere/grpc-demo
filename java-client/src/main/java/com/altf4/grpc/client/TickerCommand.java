package com.altf4.grpc.client;

import io.grpc.Context;

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
        try {
            client.ticker(TimeZone.getDefault(), intervalNanos, (String message) -> {
                System.out.println(message);
            });
        } catch (InterruptedException e) {
            System.err.println(e.getMessage());
        }
    }
}
