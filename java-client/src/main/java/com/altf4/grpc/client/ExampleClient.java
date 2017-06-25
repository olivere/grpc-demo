package com.altf4.grpc.client;

import com.altf4.grpc.*;
import io.grpc.CallCredentials;
import io.grpc.Context;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.stub.StreamObserver;

import java.util.TimeZone;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * ExampleClient represents a client for using the Example service.
 */
public class ExampleClient {
    private static final Logger logger = Logger.getLogger(ExampleClient.class.getName());

    public static final int MAX_INBOUND_MESSAGE_SIZE = 1 << 20;
    public static final int MAX_OUTBOUND_MESSAGE_SIZE = 1 << 20;

    private final ManagedChannel channel;
    private final CallCredentials callCredentials;

    public ExampleClient(String host, int port, CallCredentials callCredentials) {
        this(ManagedChannelBuilder.forAddress(host, port), callCredentials);
    }

    public ExampleClient(ManagedChannelBuilder<?> channelBuilder, CallCredentials callCredentials) {
        this.channel = channelBuilder.build();
        this.callCredentials = callCredentials;
    }

    /**
     * Shuts down the communication channel to the server.
     * It waits for a graceful shutdown of 5 seconds until it is enforced.
     *
     * @throws InterruptedException
     */
    public void shutdown() throws InterruptedException {
        channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
    }

    /**
     * Creates a blocking stub and initializes it with the generic channel parameters
     * like message size and authentication settings. The caller might set additional
     * options.
     *
     * @return Blocking stub
     */
    private ExampleGrpc.ExampleBlockingStub createBlockingStub() {
        ExampleGrpc.ExampleBlockingStub stub = ExampleGrpc.newBlockingStub(channel)
                .withMaxInboundMessageSize(MAX_INBOUND_MESSAGE_SIZE)
                .withMaxOutboundMessageSize(MAX_OUTBOUND_MESSAGE_SIZE);
        if (callCredentials != null) {
            stub = stub.withCallCredentials(callCredentials);
        }
        return stub;
    }

    /**
     * Creates an async stub and initializes it with the generic channel parameters
     * like message size and authentication settings. The caller might set additional
     * options.
     *
     * @return Asynchronous stub
     */
    private ExampleGrpc.ExampleStub createAsyncStub() {
        ExampleGrpc.ExampleStub stub = ExampleGrpc.newStub(channel)
                .withMaxInboundMessageSize(MAX_INBOUND_MESSAGE_SIZE)
                .withMaxOutboundMessageSize(MAX_OUTBOUND_MESSAGE_SIZE);
        if (callCredentials != null) {
            stub = stub.withCallCredentials(callCredentials);
        }
        return stub;
    }

    /**
     * Calls the Hello RPC, and returns its response.

     * @param request Request to initiate the call with
     * @return Response from the RPC call
     */
    public HelloResponse hello(HelloRequest request) throws InterruptedException {
        ExampleGrpc.ExampleStub stub = createAsyncStub().withDeadlineAfter(10, TimeUnit.SECONDS);
        // return stub.hello(request);

        final HelloResponse[] helloResponse = {null};
        CountDownLatch finishLatch = new CountDownLatch(1);
        stub.hello(request, new StreamObserver<HelloResponse>() {
            @Override
            public void onNext(HelloResponse response) {
                helloResponse[0] = response;
            }

            @Override
            public void onError(Throwable throwable) {
                finishLatch.countDown();
            }

            @Override
            public void onCompleted() {
                finishLatch.countDown();
            }
        });
        finishLatch.await();
        return helloResponse[0];
    }

    public void ticker(TimeZone tz, long intervalNanos, Consumer<String> callback) throws InterruptedException {
        ExampleGrpc.ExampleStub stub = createAsyncStub();

        TickerRequest request = TickerRequest.newBuilder()
                .setTimezone(tz.getID())
                .setInterval(intervalNanos)
                .build();

        final CountDownLatch finishLatch = new CountDownLatch(1);
        StreamObserver<TickerResponse> responseObserver = new StreamObserver<TickerResponse>() {
            @Override
            public void onNext(TickerResponse tickerResponse) {
                callback.accept(tickerResponse.getTick());
            }

            @Override
            public void onError(Throwable t) {
                logger.log(Level.WARNING, "ticker failed: " + t.getMessage());
                finishLatch.countDown();
            }

            @Override
            public void onCompleted() {
                finishLatch.countDown();
            }
        };

        stub.ticker(request, responseObserver);
        finishLatch.await();
    }
}
