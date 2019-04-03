package com.yahoo.vespa.hosted.controller.restapi.filter;

import com.yahoo.vespa.hosted.controller.role.RoleMembership;

import java.security.Principal;

/**
 * A {@link Principal} with a {@link RoleMembership}.
 *
 * @author jonmv
 */
public interface RolePrincipal extends Principal {

    /** Returns the roles with context this principal is a member of. */
    RoleMembership roles();

}
