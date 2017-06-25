package com.altf4.grpc.client;

import com.altf4.grpc.Gender;
import com.altf4.grpc.HelloRequest;
import com.altf4.grpc.HelloResponse;
import io.grpc.Context;

import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;

/**
 * Created by oliver on 25.06.17.
 */
public class HelloCommand {
    private final ExampleClient client;

    public HelloCommand(ExampleClient client) {
        this.client = client;
    }

    public void run() throws InterruptedException {
        HelloRequest helloRequest = HelloRequest.newBuilder()
                .setName("Oliver")
                .setAge(23)
                .setGender(Gender.MALE)
                .setOnline(true)
                .build();
        HelloResponse helloResponse = client.hello(helloRequest);
        if (helloResponse != null)
            System.out.println(helloResponse.getMessage());
    }
}
