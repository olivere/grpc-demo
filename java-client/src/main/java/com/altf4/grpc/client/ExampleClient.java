package com.altf4.grpc.client;

import com.altf4.grpc.*;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.StatusRuntimeException;

import java.util.Iterator;
import java.util.TimeZone;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;

/**
 * ExampleClient represents a client for using the Example service.
 */
public class ExampleClient {
    private final ManagedChannel channel;
    private final ExampleGrpc.ExampleBlockingStub blockingStub;
    private final ExampleGrpc.ExampleStub asyncStub;

    public ExampleClient(String host, int port) {
        this(ManagedChannelBuilder.forAddress(host, port));
    }

    public ExampleClient(ManagedChannelBuilder<?> channelBuilder) {
        channel = channelBuilder.build();
        blockingStub = ExampleGrpc.newBlockingStub(channel);
        asyncStub = ExampleGrpc.newStub(channel);
    }

    public void shutdown() throws InterruptedException {
        channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
    }

    public HelloResponse hello(HelloRequest request) {
        return blockingStub.hello(request);
    }

    public void ticker(TimeZone tz, long intervalNanos, Consumer<String> callback) {
        TickerRequest request = TickerRequest.newBuilder()
                .setTimezone(tz.getID())
                .setInterval(intervalNanos)
                .build();

        Iterator<TickerResponse> responses;
        try {
            responses = blockingStub.ticker(request);
            while (responses.hasNext()) {
                TickerResponse response = responses.next();
                callback.accept(response.getTick());
            }
        } catch (StatusRuntimeException e) {
        }
    }
}
