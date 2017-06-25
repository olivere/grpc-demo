package com.altf4.grpc.client;

import com.altf4.grpc.Gender;
import com.altf4.grpc.HelloRequest;
import com.altf4.grpc.HelloResponse;

/**
 * Created by oliver on 25.06.17.
 */
public class HelloCommand {
    private final ExampleClient client;

    public HelloCommand(ExampleClient client) {
        this.client = client;
    }

    public void run() {
        HelloRequest helloRequest = HelloRequest.newBuilder()
                .setName("Oliver")
                .setAge(23)
                .setGender(Gender.MALE)
                .setOnline(true)
                .build();
        HelloResponse helloResponse = client.hello(helloRequest);
        System.out.println(helloResponse.getMessage());
    }
}
