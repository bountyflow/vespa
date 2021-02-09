// Copyright Verizon Media. Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.
package com.yahoo.vespa.hosted.provision.maintenance;

import com.yahoo.config.provision.ApplicationId;
import com.yahoo.config.provision.ApplicationTransaction;
import com.yahoo.config.provision.Capacity;
import com.yahoo.config.provision.ClusterResources;
import com.yahoo.config.provision.ClusterSpec;
import com.yahoo.config.provision.Environment;
import com.yahoo.config.provision.NodeResources;
import com.yahoo.config.provision.NodeType;
import com.yahoo.config.provision.RegionName;
import com.yahoo.config.provision.Zone;
import com.yahoo.transaction.NestedTransaction;
import com.yahoo.vespa.hosted.provision.Node;
import com.yahoo.vespa.hosted.provision.NodeList;
import com.yahoo.vespa.hosted.provision.provisioning.ProvisioningTester;
import com.yahoo.vespa.hosted.provision.testutils.MockDeployer;
import com.yahoo.vespa.hosted.provision.testutils.MockDeployer.ApplicationContext;
import com.yahoo.vespa.hosted.provision.testutils.MockDeployer.ClusterContext;
import org.junit.Test;

import java.time.Duration;
import java.util.Comparator;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;
import java.util.stream.Stream;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;

/**
 * @author mpolden
 */
public class SwitchRebalancerTest {

    private static final ApplicationId app = ApplicationId.from("t1", "a1", "i1");

    @Test
    public void rebalance() {
        ClusterSpec.Id cluster1 = ClusterSpec.Id.from("c1");
        ClusterSpec.Id cluster2 = ClusterSpec.Id.from("c2");
        ProvisioningTester tester = new ProvisioningTester.Builder().zone(new Zone(Environment.prod, RegionName.from("us-east"))).build();
        MockDeployer deployer = deployer(tester, cluster1, cluster2);
        SwitchRebalancer rebalancer = new SwitchRebalancer(tester.nodeRepository(), Duration.ofDays(1), new TestMetric(), deployer);

        // Provision initial hosts on same switch
        NodeResources hostResources = new NodeResources(48, 128, 500, 10);
        List<Node> hosts0 = tester.makeReadyNodes(3, hostResources, NodeType.host, 5);
        tester.activateTenantHosts();
        String switch0 = "switch0";
        tester.patchNodes(hosts0, (host) -> host.withSwitchHostname(switch0));

        // Deploy application
        deployer.deployFromLocalActive(app).get().activate();
        tester.assertSwitches(Set.of(switch0), app, cluster1);
        tester.assertSwitches(Set.of(switch0), app, cluster2);

        // Rebalancing does nothing as there are no better moves to perform
        tester.clock().advance(SwitchRebalancer.waitTimeAfterPreviousDeployment);
        assertNoMoves(rebalancer, tester);

        // Provision hosts on distinct switches
        List<Node> hosts1 = tester.makeReadyNodes(3, hostResources, NodeType.host, 5);
        tester.activateTenantHosts();
        for (int i = 0; i < hosts1.size(); i++) {
            String switchHostname = "switch" + (i + 1);
            tester.patchNode(hosts1.get(i), (host) -> host.withSwitchHostname(switchHostname));
        }

        // Application is redeployed
        deployer.deployFromLocalActive(app).get().activate();

        // Rebalancer does nothing as not enough time has passed since previous deployment
        assertNoMoves(rebalancer, tester);

        // Rebalancer retires one node from non-exclusive switch in each cluster, and allocates a new one
        for (var cluster : List.of(cluster1, cluster2)) {
            tester.clock().advance(SwitchRebalancer.waitTimeAfterPreviousDeployment);
            rebalancer.maintain();
            NodeList allNodes = tester.nodeRepository().nodes().list();
            NodeList clusterNodes = allNodes.owner(app).cluster(cluster).state(Node.State.active);
            NodeList retired = clusterNodes.retired();
            assertEquals("Node is retired in " + cluster, 1, retired.size());
            assertEquals("Cluster " + cluster + " allocates nodes on distinct switches", 2,
                         tester.switchesOf(clusterNodes, allNodes).size());

            // Retired node becomes inactive and makes zone stable
            try (var lock = tester.provisioner().lock(app)) {
                NestedTransaction removeTransaction = new NestedTransaction();
                tester.nodeRepository().nodes().deactivate(retired.asList(), new ApplicationTransaction(lock, removeTransaction));
                removeTransaction.commit();
            }
        }

        // Next run does nothing
        tester.clock().advance(SwitchRebalancer.waitTimeAfterPreviousDeployment);
        assertNoMoves(rebalancer, tester);
    }

    @Test
    public void rebalance_does_not_move_node_already_on_exclusive_switch() {
        ProvisioningTester tester = new ProvisioningTester.Builder().zone(new Zone(Environment.prod, RegionName.from("us-east"))).build();
        ClusterSpec spec = ClusterSpec.request(ClusterSpec.Type.content, ClusterSpec.Id.from("c1")).vespaVersion("1").build();
        Capacity capacity = Capacity.from(new ClusterResources(4, 1, new NodeResources(4, 8, 50, 1)));
        MockDeployer deployer = deployer(tester, capacity, spec);
        SwitchRebalancer rebalancer = new SwitchRebalancer(tester.nodeRepository(), Duration.ofDays(1), new TestMetric(), deployer);

        // Provision initial hosts on two switches
        NodeResources hostResources = new NodeResources(8, 16, 500, 10);
        List<Node> hosts0 = tester.makeReadyNodes(4, hostResources, NodeType.host, 5);
        hosts0.sort(Comparator.comparing(Node::hostname));
        tester.activateTenantHosts();
        String switch0 = "switch0";
        String switch1 = "switch1";
        tester.patchNode(hosts0.get(0), (host) -> host.withSwitchHostname(switch0));
        tester.patchNodes(hosts0.subList(1, hosts0.size()), (host) -> host.withSwitchHostname(switch1));

        // Deploy application
        deployer.deployFromLocalActive(app).get().activate();
        tester.assertSwitches(Set.of(switch0, switch1), app, spec.id());
        List<Node> nodesOnExclusiveSwitch = tester.activeNodesOn(switch0, app, spec.id());
        assertEquals(1, nodesOnExclusiveSwitch.size());
        assertEquals(3, tester.activeNodesOn(switch1, app, spec.id()).size());

        // Another host becomes available on a new host
        List<Node> hosts2 = tester.makeReadyNodes(1, hostResources, NodeType.host, 5);
        tester.activateTenantHosts();
        String switch2 = "switch2";
        tester.patchNodes(hosts2, (host) -> host.withSwitchHostname(switch2));

        // Rebalance
        tester.clock().advance(SwitchRebalancer.waitTimeAfterPreviousDeployment);
        rebalancer.maintain();
        NodeList activeNodes = tester.nodeRepository().nodes().list().owner(app).cluster(spec.id()).state(Node.State.active);
        NodeList retired = activeNodes.retired();
        assertEquals("Node is retired", 1, retired.size());
        assertFalse("Retired node was not on exclusive switch", nodesOnExclusiveSwitch.contains(retired.first().get()));
        tester.assertSwitches(Set.of(switch0, switch1, switch2), app, spec.id());
        // Retired node becomes inactive and makes zone stable
        try (var lock = tester.provisioner().lock(app)) {
            NestedTransaction removeTransaction = new NestedTransaction();
            tester.nodeRepository().nodes().deactivate(retired.asList(), new ApplicationTransaction(lock, removeTransaction));
            removeTransaction.commit();
        }

        // Next iteration does nothing
        tester.clock().advance(SwitchRebalancer.waitTimeAfterPreviousDeployment);
        assertNoMoves(rebalancer, tester);
    }

    private void assertNoMoves(SwitchRebalancer rebalancer, ProvisioningTester tester) {
        NodeList nodes0 = tester.nodeRepository().nodes().list(Node.State.active).owner(app);
        rebalancer.maintain();
        NodeList nodes1 = tester.nodeRepository().nodes().list(Node.State.active).owner(app);
        assertEquals("Node allocation is unchanged", nodes0.asList(), nodes1.asList());
        assertEquals("No nodes are retired", List.of(), nodes1.retired().asList());
    }

    private static MockDeployer deployer(ProvisioningTester tester, ClusterSpec.Id containerCluster, ClusterSpec.Id contentCluster) {
        return deployer(tester,
                        Capacity.from(new ClusterResources(2, 1, new NodeResources(4, 8, 50, 1))),
                        ClusterSpec.request(ClusterSpec.Type.container, containerCluster).vespaVersion("1").build(),
                        ClusterSpec.request(ClusterSpec.Type.content, contentCluster).vespaVersion("1").build());
    }

    private static MockDeployer deployer(ProvisioningTester tester, Capacity capacity, ClusterSpec first, ClusterSpec... rest) {
        List<ClusterContext> clusterContexts = Stream.concat(Stream.of(first), Stream.of(rest))
                                                     .map(spec -> new ClusterContext(app, spec, capacity))
                                                     .collect(Collectors.toList());
        ApplicationContext context = new ApplicationContext(app, clusterContexts);
        return new MockDeployer(tester.provisioner(), tester.clock(), Map.of(app, context));
    }

}
