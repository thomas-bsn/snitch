from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...types import File, FileTypes
from io import BytesIO
from typing import cast



def _get_kwargs(
    id: int,
    *,
    x_worker_lease_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["X-Worker-Lease-Token"] = x_worker_lease_token







    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/worker/jobs/{id}/media".format(id=quote(str(id), safe=""),),
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | File:
    if response.status_code == 200:
        response_200 = File(
             payload = BytesIO(response.content)
        )



        return response_200

    if response.status_code == 206:
        response_206 = File(
             payload = BytesIO(response.content)
        )



        return response_206

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | File]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    *,
    client: AuthenticatedClient,
    x_worker_lease_token: str,

) -> Response[Error | File]:
    """  L’accès média exige un lease détenu par le worker. Les réponses n’exposent aucun chemin système.

    Args:
        id (int):
        x_worker_lease_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | File]
     """


    kwargs = _get_kwargs(
        id=id,
x_worker_lease_token=x_worker_lease_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    *,
    client: AuthenticatedClient,
    x_worker_lease_token: str,

) -> Error | File | None:
    """  L’accès média exige un lease détenu par le worker. Les réponses n’exposent aucun chemin système.

    Args:
        id (int):
        x_worker_lease_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | File
     """


    return sync_detailed(
        id=id,
client=client,
x_worker_lease_token=x_worker_lease_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient,
    x_worker_lease_token: str,

) -> Response[Error | File]:
    """  L’accès média exige un lease détenu par le worker. Les réponses n’exposent aucun chemin système.

    Args:
        id (int):
        x_worker_lease_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | File]
     """


    kwargs = _get_kwargs(
        id=id,
x_worker_lease_token=x_worker_lease_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    *,
    client: AuthenticatedClient,
    x_worker_lease_token: str,

) -> Error | File | None:
    """  L’accès média exige un lease détenu par le worker. Les réponses n’exposent aucun chemin système.

    Args:
        id (int):
        x_worker_lease_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | File
     """


    return (await asyncio_detailed(
        id=id,
client=client,
x_worker_lease_token=x_worker_lease_token,

    )).parsed
