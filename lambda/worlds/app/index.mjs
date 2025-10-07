import { S3Client, ListObjectsV2Command } from "@aws-sdk/client-s3";

const region = process.env.AWS_REGION || "us-east-2";
const bucket = process.env.WORLD_BUCKET;
const corsOrigin = process.env.CORS_ORIGIN || "*";

const s3 = new S3Client({ region });

export const handler = async () => {
    try {
        if (!bucket) {
            return jsonResponse(500, { error: "WORLD_BUCKET not configured" });
        }

        const res = await s3.send(
            new ListObjectsV2Command({
                Bucket: bucket,
                Prefix: "worlds/",
                Delimiter: "/",
            })
        );

        const worlds = (res.CommonPrefixes || []).map((p) => ({
            id: p.Prefix.replace(/^worlds\//, "").replace(/\/$/, ""),
            url: `https://${bucket}.s3.${region}.amazonaws.com/${p.Prefix}`,
        }));

        return jsonResponse(200, worlds);
    } catch (err) {
        console.error("Error listing worlds:", err);
        return jsonResponse(500, { error: "Failed to list worlds" });
    }
};

function jsonResponse(statusCode, body) {
    return {
        statusCode,
        headers: {
            "Access-Control-Allow-Origin": corsOrigin,
            "Access-Control-Allow-Headers": "Content-Type",
            "Content-Type": "application/json",
        },
        body: JSON.stringify(body),
    };
}
