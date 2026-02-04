fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Compile proto files for gRPC client
    tonic_build::configure()
        .build_server(false)  // We only need the client
        .compile_protos(
            &["../go-agent-service/api/proto/agent.proto"],
            &["../go-agent-service/api/proto"],
        )?;
    Ok(())
}
