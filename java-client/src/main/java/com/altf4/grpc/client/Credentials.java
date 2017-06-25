package com.altf4.grpc.client;

import io.grpc.Attributes;
import io.grpc.Metadata;
import io.grpc.MethodDescriptor;
import io.grpc.Status;

import java.util.concurrent.Executor;

/**
 * Created by oliver on 25.06.17.
 */
public class Credentials implements io.grpc.CallCredentials {
    public static final Metadata.Key<String> USER_KEY = Metadata.Key.of("user", Metadata.ASCII_STRING_MARSHALLER);

    private final String user;

    public Credentials(String user) {
        this.user = user;
    }

    @Override
    public void applyRequestMetadata(MethodDescriptor<?, ?> methodDescriptor, Attributes attributes, Executor executor, MetadataApplier metadataApplier) {
        executor.execute(() -> {
            try {
                Metadata md = new Metadata();
                md.put(USER_KEY, this.user);
                metadataApplier.apply(md);
            } catch (Throwable e) {
                metadataApplier.fail(Status.UNAUTHENTICATED.withCause(e));
            }
        });
    }
}
