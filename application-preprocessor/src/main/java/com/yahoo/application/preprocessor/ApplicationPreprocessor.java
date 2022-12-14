// Copyright Yahoo. Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.
package com.yahoo.application.preprocessor;

import com.yahoo.config.application.api.ApplicationPackage;
import com.yahoo.config.model.application.provider.BaseDeployLogger;
import com.yahoo.config.model.application.provider.FilesApplicationPackage;
import com.yahoo.config.provision.Environment;
import com.yahoo.config.provision.RegionName;
import com.yahoo.config.provision.Zone;
import com.yahoo.yolean.Exceptions;
import java.io.File;
import java.io.IOException;
import java.util.Optional;

/**
 * Main entry for preprocessing an application package.
 *
 * @author Ulf Lilleengen
 */
public class ApplicationPreprocessor {

    private final File applicationDir;
    private final Optional<File> outputDir;
    private final Optional<Environment> environment;
    private final Optional<RegionName> region;

    public ApplicationPreprocessor(File applicationDir, Optional<File> outputDir, Optional<Environment> environment, Optional<RegionName> region) {
        this.applicationDir = applicationDir;
        this.outputDir = outputDir;
        this.environment = environment;
        this.region = region;
    }

    public void run() throws IOException {
        FilesApplicationPackage.Builder applicationPackageBuilder = new FilesApplicationPackage.Builder(applicationDir);
        outputDir.ifPresent(applicationPackageBuilder::preprocessedDir);
        ApplicationPackage preprocessed = applicationPackageBuilder.build().preprocess(
                new Zone(environment.orElse(Environment.defaultEnvironment()), region.orElse(RegionName.defaultName())),
                new BaseDeployLogger());
        preprocessed.validateXML();
    }

    public static void main(String[] args) {
        int argCount = args.length;
        if (argCount < 1) {
            System.out.println("Usage: vespa-application-preprocessor <application package path> [environment] [region] [output path]");
            System.exit(1);
        }
        File applicationDir = new File(args[0]);
        Optional<Environment> environment = (argCount > 1) ? Optional.of(Environment.valueOf(args[1])) : Optional.empty();
        Optional<RegionName> region = (argCount > 2) ? Optional.of(RegionName.from(args[2])) : Optional.empty();
        Optional<File> outputDir = (argCount > 3) ? Optional.of(new File(args[3])) : Optional.empty();
        ApplicationPreprocessor preprocessor = new ApplicationPreprocessor(applicationDir, outputDir, environment, region);
        try {
            preprocessor.run();
            System.out.println("Application preprocessed successfully and written to " +
                               outputDir.orElse(new File(applicationDir, FilesApplicationPackage.preprocessed)).getAbsolutePath());
        } catch (Exception e) {
            System.err.println("Error validating application package: " + Exceptions.toMessageString(e));
            System.exit(1);
        }
    }

}
