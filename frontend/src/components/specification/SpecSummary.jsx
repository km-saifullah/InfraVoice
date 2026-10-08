function ResourceRow({ label, children }) {
  return (
    <div className="flex items-center justify-between gap-3 border-b border-ink-800 py-2 text-sm last:border-0">
      <span className="text-mist-400">{label}</span>
      <span className="min-w-0 break-all text-right font-mono text-mist-100">{children}</span>
    </div>
  );
}

export function SpecSummary({ spec }) {
  if (!spec) return null;

  const vpcs = spec.vpcs || [];
  const ec2 = spec.ec2 || [];
  const s3 = spec.s3 || [];
  const sns = spec.sns || [];
  const isEmpty = vpcs.length + ec2.length + s3.length + sns.length === 0;

  if (isEmpty) {
    return (
      <p className="text-sm text-mist-400">
        This specification does not define any resources yet.
      </p>
    );
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2">
      {vpcs.map((vpc, index) => (
        <div
          key={`vpc-${index}`}
          className="rounded-md border border-ink-800 p-4"
        >
          <h4 className="font-display text-sm font-semibold text-mist-100">
            VPC · {vpc.name}
          </h4>
          <ResourceRow label="CIDR">{vpc.cidr}</ResourceRow>
          <ResourceRow label="Subnets">
            {(vpc.subnets || []).length}
          </ResourceRow>
          {(vpc.subnets || []).map((subnet, subnetIndex) => (
            <div key={subnetIndex} className="mt-1 pl-3 text-xs text-mist-500">
              {subnet.name} · {subnet.cidr} · {subnet.type}
            </div>
          ))}
        </div>
      ))}

      {ec2.map((instance, index) => (
        <div
          key={`ec2-${index}`}
          className="rounded-md border border-ink-800 p-4"
        >
          <h4 className="font-display text-sm font-semibold text-mist-100">
            EC2 · {instance.name}
          </h4>
          <ResourceRow label="Type">{instance.instance_type}</ResourceRow>
          <ResourceRow label="AMI">{instance.ami || "not set"}</ResourceRow>
          <ResourceRow label="Subnet">
            {instance.subnet || "default"}
          </ResourceRow>
          <ResourceRow label="Root volume">
            {instance.root_volume_gb} GB
          </ResourceRow>
          <ResourceRow label="Count">{instance.count}</ResourceRow>
        </div>
      ))}

      {s3.map((bucket, index) => (
        <div
          key={`s3-${index}`}
          className="rounded-md border border-ink-800 p-4"
        >
          <h4 className="font-display text-sm font-semibold text-mist-100">
            S3 · {bucket.name}
          </h4>
          <ResourceRow label="Bucket name">
            {bucket.bucket_name || "auto-generated"}
          </ResourceRow>
          <ResourceRow label="Versioning">
            {bucket.versioning ? "on" : "off"}
          </ResourceRow>
          <ResourceRow label="Encryption">
            {bucket.encryption ? "on" : "off"}
          </ResourceRow>
        </div>
      ))}

      {sns.map((topic, index) => (
        <div
          key={`sns-${index}`}
          className="rounded-md border border-ink-800 p-4"
        >
          <h4 className="font-display text-sm font-semibold text-mist-100">
            SNS · {topic.name}
          </h4>
          <ResourceRow label="Display name">
            {topic.display_name || "—"}
          </ResourceRow>
        </div>
      ))}
    </div>
  );
}
