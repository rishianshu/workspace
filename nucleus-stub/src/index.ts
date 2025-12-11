
import { ApolloServer } from '@apollo/server';
import { startStandaloneServer } from '@apollo/server/standalone';
import { readFileSync } from 'fs';
import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import path from 'path';

// --- Data Layer (Mock Seeds) ---
const nodes = require('./seeds/kp_nodes.json');
const edges = require('./seeds/kp_edges.json');

console.log(`[Stub] Loaded ${nodes.length} nodes and ${edges.length} edges.`);

// --- GraphQL Server ---
const typeDefs = readFileSync(path.join(__dirname, 'schema.graphql'), 'utf-8');

const resolvers = {
  Query: {
    // Universal Search
    search: (_: any, args: { query: string; filters?: any }) => {
      const q = args.query.toLowerCase();
      const results = nodes
        .filter((n: any) => 
          n.displayName.toLowerCase().includes(q) || 
          n.canonicalPath.toLowerCase().includes(q)
        )
        .map((n: any) => ({
          node: n,
          score: 1.0,
          highlights: []
        }));
      return results;
    },
    
    // Entity Resolution
    entity: (_: any, args: { id: string }) => {
      return nodes.find((n: any) => n.id === args.id);
    },

    entityBySource: (_: any, args: { system: string; key: string }) => {
      return nodes.find((n: any) => 
        n.sourceSystem === args.system && n.canonicalPath === args.key
      );
    }
  },
  
  GraphNode: {
    // Relationship Traversal
    edges: (parent: any, args: { type?: string; direction?: string }) => {
        // Simple adjacency lookup
        return edges.filter((e: any) => 
            e.sourceId === parent.id || e.targetId === parent.id
        ).map((e: any) => {
            // naive logic: if source is me, target is the other node
            const targetId = e.sourceId === parent.id ? e.targetId : e.sourceId;
            const target = nodes.find((n: any) => n.id === targetId);
            return {
                id: e.id,
                edgeType: e.relationshipType, // Map relationshipType to edgeType
                target: target,
                metadata: e.properties
            };
        }).filter((e: any) => !!e.target); // filter broken links
    }
  },

  Mutation: {
    executeAction: (_: any, args: { nodeId: string; verb: string; payload: any }) => {
      console.log(`[Stub][Mutation] executeAction: ${args.verb} on ${args.nodeId}`, args.payload);
      
      // In a real implementation this would call the gRPC Execute()
      // For stub, we just pretend.
      
      const node = nodes.find((n: any) => n.id === args.nodeId);
      if (node) {
          // Optimistic update for stub demo
          if (args.payload.status) node.properties.status = args.payload.status;
      }
      
      return {
        success: true,
        message: `Executed ${args.verb}`,
        updatedNode: node
      };
    }
  }
};

async function startGraphQL() {
  const server = new ApolloServer({
    typeDefs,
    resolvers,
  });

  const { url } = await startStandaloneServer(server, {
    listen: { port: 4000 },
  });

  console.log(`🚀 GraphQL Brain ready at: ${url}`);
}

// --- gRPC Server (UCL) ---
const PROTO_PATH = path.join(__dirname, 'protos/connector.proto');
const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true
});

const connectorProto = grpc.loadPackageDefinition(packageDefinition) as any;

function startGRPC() {
    const server = new grpc.Server();
    
    // Implement ucl.connector.v1.UnimplementedConnectorService? 
    // The proto likely defines 'ConnectorService'.
    const uclPackage = connectorProto.ucl.connector.v1;
    
    server.addService(uclPackage.ConnectorService.service, {
        ValidateConfig: (call: any, callback: any) => {
             callback(null, { valid: true, message: "Valid stub config" });
        },
        ListDatasets: (call: any, callback: any) => {
            console.log("[Stub][gRPC] ListDatasets called");
            callback(null, { 
                datasets: [
                    { id: "tickets", name: "Jira Tickets", kind: "table" },
                    { id: "prs", name: "GitHub PRs", kind: "stream" }
                 ] 
            });
        },
        Execute: (call: any, callback: any) => {
            const req = call.request;
            console.log(`[Stub][gRPC] Execute Action: ${req.action}`, req.parameters);
            callback(null, { result: { success: true } });
        },
        // Stream dummy data
        Read: (call: any) => {
            console.log("[Stub][gRPC] Read stream started");
            nodes.forEach((n: any) => {
                call.write({ record: n });
            });
            call.end();
        }
    });
    
    server.bindAsync('0.0.0.0:50051', grpc.ServerCredentials.createInsecure(), () => {
        // server.start(); // gRPC 1.10+ starts automatically or via logic? 
        // Update: in recent grpc-js, verify if start() is needed. Ideally it is not constrained.
        // But commonly checking documentation: server.start() is deprecated in some versions but good to check.
        // Actually bindAsync does not auto-start.
        console.log(`🔗 gRPC UCL ready at: 0.0.0.0:50051`);
    });
}

// --- Main ---
async function main() {
    await startGraphQL();
    startGRPC();
}

main();
