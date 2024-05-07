import { z } from "zod";

import {
  GroupsSchema,
  IncidentContactsSchema,
  OrganizationsSchema,
  OrgMembershipsSchema,
  OrgRolesSchema,
  UsersSchema
} from "@app/db/schemas";
import { ORGANIZATIONS } from "@app/lib/api-docs";
import { readLimit, writeLimit } from "@app/server/config/rateLimiter";
import { verifyAuth } from "@app/server/plugins/auth/verify-auth";
import { AuthMode } from "@app/services/auth/auth-type";

export const registerOrgRouter = async (server: FastifyZodProvider) => {
  server.route({
    method: "GET",
    url: "/",
    config: {
      rateLimit: readLimit
    },
    schema: {
      response: {
        200: z.object({
          organizations: OrganizationsSchema.array()
        })
      }
    },
    onRequest: verifyAuth([AuthMode.JWT], { requireOrg: false }),
    handler: async (req) => {
      const organizations = await server.services.org.findAllOrganizationOfUser(req.permission.id);
      return { organizations };
    }
  });

  server.route({
    method: "GET",
    url: "/:organizationId",
    config: {
      rateLimit: readLimit
    },
    schema: {
      params: z.object({
        organizationId: z.string().trim()
      }),
      response: {
        200: z.object({
          organization: OrganizationsSchema
        })
      }
    },
    onRequest: verifyAuth([AuthMode.JWT]),
    handler: async (req) => {
      const organization = await server.services.org.findOrganizationById(
        req.permission.id,
        req.params.organizationId,
        req.permission.authMethod,
        req.permission.orgId
      );
      return { organization };
    }
  });

  server.route({
    method: "GET",
    url: "/:organizationId/users",
    config: {
      rateLimit: readLimit
    },
    schema: {
      params: z.object({
        organizationId: z.string().trim()
      }),
      response: {
        200: z.object({
          users: OrgMembershipsSchema.merge(
            z.object({
              user: UsersSchema.pick({
                username: true,
                email: true,
                firstName: true,
                lastName: true,
                id: true
              }).merge(z.object({ publicKey: z.string().nullable() }))
            })
          )
            .omit({ createdAt: true, updatedAt: true })
            .array()
        })
      }
    },
    onRequest: verifyAuth([AuthMode.JWT]),
    handler: async (req) => {
      const users = await server.services.org.findAllOrgMembers(
        req.permission.id,
        req.params.organizationId,
        req.permission.authMethod,
        req.permission.orgId
      );
      return { users };
    }
  });

  server.route({
    method: "PATCH",
    url: "/:organizationId",
    config: {
      rateLimit: writeLimit
    },
    schema: {
      params: z.object({ organizationId: z.string().trim() }),
      body: z.object({
        name: z.string().trim().max(64, { message: "Name must be 64 or fewer characters" }).optional(),
        slug: z
          .string()
          .trim()
          .max(64, { message: "Slug must be 64 or fewer characters" })
          .regex(/^[a-zA-Z0-9-]+$/, "Slug must only contain alphanumeric characters or hyphens")
          .optional(),
        authEnforced: z.boolean().optional(),
        scimEnabled: z.boolean().optional()
      }),
      response: {
        200: z.object({
          message: z.string(),
          organization: OrganizationsSchema
        })
      }
    },
    onRequest: verifyAuth([AuthMode.JWT]),
    handler: async (req) => {
      const organization = await server.services.org.updateOrg({
        actor: req.permission.type,
        actorId: req.permission.id,
        actorOrgId: req.permission.orgId,
        actorAuthMethod: req.permission.authMethod,
        orgId: req.params.organizationId,
        data: req.body
      });

      return {
        message: "Successfully changed organization name",
        organization
      };
    }
  });

  server.route({
    method: "GET",
    url: "/:organizationId/incidentContactOrg",
    config: {
      rateLimit: readLimit
    },
    schema: {
      params: z.object({ organizationId: z.string().trim() }),
      response: {
        200: z.object({
          incidentContactsOrg: IncidentContactsSchema.array()
        })
      }
    },
    onRequest: verifyAuth([AuthMode.JWT]),
    handler: async (req) => {
      const incidentContactsOrg = await req.server.services.org.findIncidentContacts(
        req.permission.id,
        req.params.organizationId,
        req.permission.authMethod,
        req.permission.orgId
      );
      return { incidentContactsOrg };
    }
  });

  server.route({
    method: "POST",
    url: "/:organizationId/incidentContactOrg",
    config: {
      rateLimit: writeLimit
    },
    schema: {
      params: z.object({ organizationId: z.string().trim() }),
      body: z.object({ email: z.string().email().trim() }),
      response: {
        200: z.object({
          incidentContactsOrg: IncidentContactsSchema
        })
      }
    },
    onRequest: verifyAuth([AuthMode.JWT]),
    handler: async (req) => {
      const incidentContactsOrg = await req.server.services.org.createIncidentContact(
        req.permission.id,
        req.params.organizationId,
        req.body.email,
        req.permission.authMethod,
        req.permission.orgId
      );
      return { incidentContactsOrg };
    }
  });

  server.route({
    method: "DELETE",
    url: "/:organizationId/incidentContactOrg/:incidentContactId",
    config: {
      rateLimit: writeLimit
    },
    schema: {
      params: z.object({ organizationId: z.string().trim(), incidentContactId: z.string().trim() }),
      response: {
        200: z.object({
          incidentContactsOrg: IncidentContactsSchema
        })
      }
    },
    onRequest: verifyAuth([AuthMode.JWT]),
    handler: async (req) => {
      const incidentContactsOrg = await req.server.services.org.deleteIncidentContact(
        req.permission.id,
        req.params.organizationId,
        req.params.incidentContactId,
        req.permission.authMethod,
        req.permission.orgId
      );
      return { incidentContactsOrg };
    }
  });

  server.route({
    method: "GET",
    url: "/:organizationId/groups",
    schema: {
      params: z.object({
        organizationId: z.string().trim().describe(ORGANIZATIONS.LIST_GROUPS.organizationId)
      }),
      response: {
        200: z.object({
          groups: GroupsSchema.merge(
            z.object({
              customRole: OrgRolesSchema.pick({
                id: true,
                name: true,
                slug: true,
                permissions: true,
                description: true
              }).optional()
            })
          ).array()
        })
      }
    },
    onRequest: verifyAuth([AuthMode.JWT]),
    handler: async (req) => {
      const groups = await server.services.org.getOrgGroups({
        actor: req.permission.type,
        actorId: req.permission.id,
        orgId: req.params.organizationId,
        actorAuthMethod: req.permission.authMethod,
        actorOrgId: req.permission.orgId
      });

      return { groups };
    }
  });
};
