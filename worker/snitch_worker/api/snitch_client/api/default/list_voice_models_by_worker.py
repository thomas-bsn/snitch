from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.worker_models_view import WorkerModelsView
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/voice-models/workers",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | WorkerModelsView:
    if response.status_code == 200:
        response_200 = WorkerModelsView.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | WorkerModelsView]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | WorkerModelsView]:
    """  Administrateurs uniquement. Vue « Modèles par Worker » : pour chaque Worker non révoqué, le modèle
    de chacun des trois types (diarisation, empreinte, séparation) avec son état réel, sa progression,
    le volume occupé, la cause d'échec et la révision.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerModelsView]
     """


    kwargs = _get_kwargs(

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,

) -> Error | WorkerModelsView | None:
    """  Administrateurs uniquement. Vue « Modèles par Worker » : pour chaque Worker non révoqué, le modèle
    de chacun des trois types (diarisation, empreinte, séparation) avec son état réel, sa progression,
    le volume occupé, la cause d'échec et la révision.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerModelsView
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | WorkerModelsView]:
    """  Administrateurs uniquement. Vue « Modèles par Worker » : pour chaque Worker non révoqué, le modèle
    de chacun des trois types (diarisation, empreinte, séparation) avec son état réel, sa progression,
    le volume occupé, la cause d'échec et la révision.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerModelsView]
     """


    kwargs = _get_kwargs(

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,

) -> Error | WorkerModelsView | None:
    """  Administrateurs uniquement. Vue « Modèles par Worker » : pour chaque Worker non révoqué, le modèle
    de chacun des trois types (diarisation, empreinte, séparation) avec son état réel, sa progression,
    le volume occupé, la cause d'échec et la révision.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerModelsView
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
