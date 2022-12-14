// Copyright Yahoo. Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.
package com.yahoo.config.subscription.impl;

import com.yahoo.config.ConfigInstance;
import com.yahoo.config.ConfigurationRuntimeException;
import com.yahoo.config.subscription.CfgConfigPayloadBuilder;
import com.yahoo.config.subscription.ConfigInterruptedException;
import com.yahoo.io.IOUtils;
import com.yahoo.vespa.config.ConfigKey;
import com.yahoo.vespa.config.ConfigPayload;

import java.io.File;
import java.io.IOException;
import java.util.Arrays;

import static java.util.logging.Level.FINE;

/**
 * Subscription used when config id is file:...
 *
 * @author Vegard Havdal
 */
public class FileConfigSubscription<T extends ConfigInstance> extends ConfigSubscription<T> {

    final File file;
    long ts;

    FileConfigSubscription(ConfigKey<T> key, File f) {
        super(key);
        setGeneration(0L);
        file = f;
        if (!file.exists() && !file.isFile())
            throw new IllegalArgumentException("Not a file: " + file);
    }

    @Override
    public boolean nextConfig(long timeout) {
        if (!file.exists() && !file.isFile()) throw new IllegalArgumentException("Not a file: " + file);
        if (checkReloaded()) {
            log.log(FINE, () -> "User forced config reload at " + System.currentTimeMillis());
            // User forced reload
            setConfigIfChanged(updateConfig());
            ConfigState<T> configState = getConfigState();
            log.log(FINE, () -> "Config updated at " + System.currentTimeMillis() + ", changed: " + configState.isConfigChanged());
            log.log(FINE, () -> "Config: " + configState.getConfig().toString());
            return true;
        }
        if (file.lastModified() != ts) {
            setConfigIncGen(updateConfig());
            return true;
        }
        try {
            Thread.sleep(timeout);
        } catch (InterruptedException e) {
            throw new ConfigInterruptedException(e);
        }

        return false;
    }

    private T updateConfig() {
        ts = file.lastModified();
        try {
            ConfigPayload payload = new CfgConfigPayloadBuilder().deserialize(Arrays.asList(IOUtils.readFile(file).split("\n")));
            return payload.toInstance(configClass, key.getConfigId());
        } catch (IOException e) {
            throw new ConfigurationRuntimeException(e);
        }
    }

    @Override
    public boolean subscribe(long timeout) {
        return true;
    }

}
